package worker

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
	"github.com/xenn00/chat-system/internal/queue"
)

func (wp *WorkerPool) StartDLQWorker(ctx context.Context) {
	wp.wg.Add(1)
	go func() {
		defer wp.wg.Done()

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

				// later we can save job.payload to DB for audit, notif trigger, etc
			}
		}
	}()
}
