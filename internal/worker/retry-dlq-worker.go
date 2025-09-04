package worker

import (
	"context"
	"encoding/json"
	"math"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/xenn00/chat-system/internal/entity"
	"github.com/xenn00/chat-system/internal/queue"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func (wp *WorkerPool) StartDLQRetryConsumer(ctx context.Context) {
	wp.wg.Add(1)
	go func() {
		defer wp.wg.Done()

		log.Info().Msg("DLQ retry consumer started")
		ticker := time.NewTicker(wp.DLQConfig.RetryInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				log.Info().Msg("DLQ retry consumer stopping")
				return
			case <-ticker.C:
				wp.processDLQJobs(ctx)
			}
		}
	}()
}

func (wp *WorkerPool) processDLQJobs(ctx context.Context) {
	collection := wp.Mongo.Database(wp.DLQConfig.DatabaseName).Collection(wp.DLQConfig.CollectionName)

	// find jobs ready for retry
	filter := bson.M{
		"status":      bson.M{"$in": []string{"pending", "failed"}},
		"retry_count": bson.M{"$lt": wp.DLQConfig.MaxRetryCount},
		"$or": []bson.M{
			{"next_retry_at": bson.M{"$exists": false}},
			{"next_retry_at": bson.M{"$lte": time.Now().UTC()}},
		},
	}

	opts := options.Find().SetSort(bson.M{"created_at": 1}).SetLimit(int64(wp.DLQConfig.BatchSize))

	cursor, err := collection.Find(ctx, filter, opts)
	if err != nil {
		log.Error().Err(err).Msg("Failed to query DLQ jobs")
		return
	}

	defer cursor.Close(ctx)

	var dlqJobs []entity.DLQJob

	if err := cursor.All(ctx, &dlqJobs); err != nil {
		log.Error().Err(err).Msg("Failed to decode DLQ jobs")
		return
	}

	if len(dlqJobs) == 0 {
		log.Debug().Msg("No DLQ jobs to process")
		return
	}

	log.Info().Int("count", len(dlqJobs)).Msg("Processing DLQ jobs")

	for _, dlqJob := range dlqJobs {
		wp.retryDLQJob(ctx, collection, &dlqJob)
	}
}

func (wp *WorkerPool) retryDLQJob(ctx context.Context, collection *mongo.Collection, dlqJob *entity.DLQJob) {
	// Update status to processing
	_, err := collection.UpdateOne(ctx, bson.M{"_id": dlqJob.ID}, bson.M{"$set": bson.M{
		"status":     "processing",
		"updated_at": time.Now().UTC(),
	}})
	if err != nil {
		log.Error().Err(err).Str("job_id", dlqJob.JobID).Msg("Failed to update DLQ job status")
		return
	}

	var originalJob queue.Job
	if err := json.Unmarshal(dlqJob.Payload, &originalJob); err != nil {
		log.Error().Err(err).Str("job_id", dlqJob.JobID).Msg("Failed to unmarshal job payload")
		wp.markDLQJobAsFailed(ctx, collection, dlqJob.ID, "invalid_payload", err.Error())
		return
	}

	// Reset retry count for fresh retry attempt
	originalJob.Retry = 0
	originalJob.ErrorMsg = ""

	// Try to process the job using existing HandleJob function
	if err := HandleJob(ctx, originalJob, wp.Redis, wp.ws); err != nil {
		// Job failed again - update retry info
		wp.handleDLQRetryFailure(ctx, collection, dlqJob, err.Error())
		return
	}

	// Job succeeded - mark as completed
	wp.markDLQJobAsCompleted(ctx, collection, dlqJob.ID)

	log.Info().Str("job_id", dlqJob.JobID).Str("type", dlqJob.Type).Int("dlq_retry_count", dlqJob.RetryCount).Msg("✅ DLQ job successfully retried")
}

func (wp *WorkerPool) handleDLQRetryFailure(ctx context.Context, collection *mongo.Collection, dlqJob *entity.DLQJob, errorMsg string) {
	newRetryCount := dlqJob.RetryCount + 1

	if newRetryCount >= wp.DLQConfig.MaxRetryCount {
		wp.markDLQJobAsPermanentlyFailed(ctx, collection, dlqJob.ID, errorMsg)

		log.Error().Str("job_id", dlqJob.JobID).Str("type", dlqJob.Type).Int("dlq_retry_count", newRetryCount).Msg("🚫 DLQ job permanently failed after max retries")
		return
	}

	// Calculate next retry time using exponential backoff
	backoffDuration := time.Duration(float64(wp.DLQConfig.RetryInterval) *
		math.Pow(wp.DLQConfig.BackoffFactor, float64(newRetryCount)))
	nextRetryAt := time.Now().UTC().Add(backoffDuration)

	// Update for next retry
	_, err := collection.UpdateOne(ctx,
		bson.M{"_id": dlqJob.ID},
		bson.M{
			"$set": bson.M{
				"status":        "failed",
				"retry_count":   newRetryCount,
				"error_msg":     errorMsg,
				"next_retry_at": nextRetryAt,
				"updated_at":    time.Now().UTC(),
			},
		},
	)

	if err != nil {
		log.Error().Err(err).Str("job_id", dlqJob.JobID).Msg("Failed to update DLQ job retry info")
		return
	}

	log.Warn().
		Str("job_id", dlqJob.JobID).
		Str("type", dlqJob.Type).
		Int("dlq_retry_count", newRetryCount).
		Time("next_retry_at", nextRetryAt).
		Msg("⏰ DLQ job scheduled for retry")
}

func (wp *WorkerPool) markDLQJobAsCompleted(ctx context.Context, collection *mongo.Collection, jobID primitive.ObjectID) {
	_, err := collection.UpdateOne(ctx,
		bson.M{"_id": jobID},
		bson.M{
			"$set": bson.M{
				"status":       "completed",
				"completed_at": time.Now().UTC(),
				"updated_at":   time.Now().UTC(),
			},
		},
	)
	if err != nil {
		log.Error().Err(err).Interface("job_id", jobID).Msg("Failed to mark DLQ job as completed")
	}
}

func (wp *WorkerPool) markDLQJobAsPermanentlyFailed(ctx context.Context, collection *mongo.Collection, jobID primitive.ObjectID, errorMsg string) {
	_, err := collection.UpdateOne(ctx,
		bson.M{"_id": jobID},
		bson.M{
			"$set": bson.M{
				"status":     "permanently_failed",
				"error_msg":  errorMsg,
				"failed_at":  time.Now().UTC(),
				"updated_at": time.Now().UTC(),
			},
		},
	)
	if err != nil {
		log.Error().Err(err).Interface("job_id", jobID).Msg("Failed to mark DLQ job as permanently failed")
	}
}

func (wp *WorkerPool) markDLQJobAsFailed(ctx context.Context, collection *mongo.Collection, jobID primitive.ObjectID, reason, errorMsg string) {
	_, err := collection.UpdateOne(ctx,
		bson.M{"_id": jobID},
		bson.M{
			"$set": bson.M{
				"status":     "failed",
				"reason":     reason,
				"error_msg":  errorMsg,
				"updated_at": time.Now().UTC(),
			},
		},
	)
	if err != nil {
		log.Error().Err(err).Interface("job_id", jobID).Msg("Failed to mark DLQ job as failed")
	}
}
