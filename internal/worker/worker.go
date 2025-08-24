package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
	"github.com/xenn00/chat-system/internal/queue"
	"github.com/xenn00/chat-system/internal/websocket"
)

type WorkerPool struct {
	Redis      *redis.Client
	WorkerNum  int
	JobChannel chan string
	wg         sync.WaitGroup
	ws         *websocket.Hub
}

func NewWorkerPool(redis *redis.Client, workerNum int, ws *websocket.Hub) *WorkerPool {
	return &WorkerPool{
		Redis:      redis,
		WorkerNum:  workerNum,
		JobChannel: make(chan string, 100), // Buffered channel to hold jobs
		ws:         ws,
	}
}

func (wp *WorkerPool) Start(ctx context.Context) {
	log.Info().Msgf("Starting worker pool with %d workers", wp.WorkerNum)

	for i := 0; i < wp.WorkerNum; i++ {
		wp.wg.Add(1)
		go wp.worker(ctx, i)
	}

	go func() {
		defer close(wp.JobChannel)
		for {
			select {
			case <-ctx.Done():
				log.Info().Msg("Stopping worker pool")
				return
			default:
				now := float64(time.Now().Unix())
				result, err := wp.Redis.ZRangeByScore(ctx, "priority_queue", &redis.ZRangeBy{
					Min:    "-inf",
					Max:    fmt.Sprintf("%f", now),
					Offset: 0,
					Count:  1,
				}).Result()

				if err != nil {
					if err != redis.Nil {
						log.Error().Err(err).Msg("Worker: failed to pop job")
					}
					continue
				}

				if len(result) == 0 {
					time.Sleep(1 * time.Second)
					continue
				}

				payload := result[0]
				wp.Redis.ZRem(ctx, "priority_queue", payload)
				wp.JobChannel <- payload
			}
		}
	}()
}

func (wp *WorkerPool) worker(ctx context.Context, id int) {
	defer wp.wg.Done()
	log.Info().Msgf("Worker %d started", id)

	for {
		select {
		case <-ctx.Done():
			log.Info().Msgf("Worker %d stopping", id)
			return
		case payload, ok := <-wp.JobChannel:
			if !ok {
				return
			}

			var job queue.Job
			if err := json.Unmarshal([]byte(payload), &job); err != nil {
				log.Warn().Err(err).Msgf("Worker %d: Failed to unmarshal job payload", id)
				continue
			}
			if err := HandleJob(ctx, job, wp.Redis, wp.ws); err != nil {
				job.Retry++
				job.ErrorMsg = err.Error()

				now := time.Now().Unix()
				if job.Retry >= job.MaxRetry || now > job.ExpireAt {
					log.Error().Str("job_id", job.ID).Msg("Job moved to DLQ")
					dlqBytes, _ := json.Marshal(job)
					wp.Redis.RPush(ctx, "priority_queue_dlq", dlqBytes)

					// Dead Letter Alert
					sendDLA(job)
				} else {
					// retry with backoff
					delay := time.Duration(5*(1<<job.Retry)) * time.Second // exponential backoff
					retryAt := time.Now().Add(delay).Unix()

					jobBytes, _ := json.Marshal(job)
					wp.Redis.ZAdd(ctx, "priority_queue", redis.Z{
						Score:  float64(retryAt),
						Member: jobBytes,
					})
					log.Warn().Str("job_id", job.ID).Msgf("Retrying in %v seconds (%d/%d)", delay.Seconds(), job.Retry, job.MaxRetry)
				}
			}
		}
	}
}

var dlaCache = make(map[string]time.Time)
var dlaMu sync.Mutex

func sendDLA(job queue.Job) {
	dlaMu.Lock()
	defer dlaMu.Unlock()

	now := time.Now()
	lastAlert, ok := dlaCache[job.Type]
	if ok && now.Sub(lastAlert) < 10*time.Minute {
		return
	}

	log.Error().Str("job_id", job.ID).Str("type", job.Type).Str("error", job.ErrorMsg).Msg("🚨 Dead Letter Alert: Job failed permanently")

	dlaCache[job.Type] = now
}

func (wp *WorkerPool) Wait() {
	wp.wg.Wait()
	log.Info().Msg("All workers have stopped")
}
