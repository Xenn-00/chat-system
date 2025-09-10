package worker

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
	"github.com/xenn00/chat-system/internal/entity"
	"github.com/xenn00/chat-system/internal/queue"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func (wp *WorkerPool) StartDLQWorker(ctx context.Context) {
	log.Info().Msg("DLQ worker started")
	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("DLQ worker stopping")
			return
		default:
			result, err := wp.Redis.BLPop(ctx, 10*time.Second, "priority_queue_dlq").Result()
			if err == redis.Nil {
				continue
			} else if err != nil {
				log.Error().Err(err).Msg("DLQWorker pop failed")
				continue
			}

			payload := result[1]
			var job queue.Job
			if err := json.Unmarshal([]byte(payload), &job); err != nil {
				log.Warn().Err(err).Msg("DLQWorker invalid job payload")
				continue
			}

			log.Error().
				Str("job_id", job.ID).
				Str("type", job.Type).
				Str("error", job.ErrorMsg).
				Msg("🚨 DLQ Job detected")

			// save to MongoDB DLQ collection
			dlqDoc := entity.DLQJob{
				JobID:              job.ID,
				Type:               job.Type,
				Payload:            job.Payload,
				Status:             "pending",
				RetryCount:         0,
				OriginalRetryCount: job.Retry,
				ErrorMsg:           job.ErrorMsg,
				CreatedAt:          time.Now().UTC(),
				ExpireAt:           time.Now().Add(7 * 24 * time.Hour).UTC(), // TTl 7 days
			}

			collection := wp.Mongo.Database(wp.DLQConfig.DatabaseName).Collection(wp.DLQConfig.CollectionName)
			if _, err := collection.InsertOne(ctx, dlqDoc); err != nil {
				log.Error().Err(err).Msg("Failed to persist DLQ job to MongoDB")

				// fallback: put back to Redis DLQ
				wp.Redis.RPush(ctx, "priority_queue_dlq", payload)
			} else {
				log.Info().Str("job_id", job.ID).Msg("DLQ job persisted to MongoDB")
			}
		}
	}
}

func (wp *WorkerPool) GetDLQStats(ctx context.Context) (map[string]int64, error) {
	collection := wp.Mongo.Database(wp.DLQConfig.DatabaseName).Collection(wp.DLQConfig.CollectionName)

	pipeline := bson.A{
		bson.M{"$group": bson.M{
			"_id":   "$status",
			"count": bson.M{"$sum": 1},
		}},
	}

	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	stats := make(map[string]int64)
	for cursor.Next(ctx) {
		var result bson.M
		if err := cursor.Decode(&result); err != nil {
			continue
		}
		status := result["_id"].(string)
		count := result["count"].(int64)
		stats[status] = count
	}

	return stats, nil
}
