package worker

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
	"github.com/xenn00/chat-system/internal/queue"
	"github.com/xenn00/chat-system/internal/utils/types"
	"github.com/xenn00/chat-system/internal/websocket"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// Lua script for atomic pop from priority queue

const atomicPopScript = `
local result = redis.call('ZRANGEBYSCORE', KEYS[1], '-inf', '+inf', 'LIMIT', 0, 1)
if #result > 0 then
	redis.call('ZREM', KEYS[1], result[1])
	return result[1]
else
	return nil
end
`

type WorkerPool struct {
	Redis      *redis.Client
	Mongo      *mongo.Client
	WorkerNum  int
	JobChannel chan string
	wg         sync.WaitGroup
	ws         *websocket.Hub
	DLQConfig  types.DLQRetryConfig

	// graceful shutdown
	ctx       context.Context
	cancel    context.CancelFunc
	atomicPop *redis.Script
}

func NewWorkerPool(redisClient *redis.Client, workerNum int, ws *websocket.Hub) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())
	return &WorkerPool{
		Redis:      redisClient,
		WorkerNum:  workerNum,
		JobChannel: make(chan string, 100), // Buffered channel to hold jobs
		ws:         ws,
		ctx:        ctx,
		cancel:     cancel,
		atomicPop:  redis.NewScript(atomicPopScript),
		DLQConfig: types.DLQRetryConfig{
			BatchSize:      10,
			RetryInterval:  5 * time.Minute,
			MaxRetryCount:  3,
			BackoffFactor:  2.0,
			DatabaseName:   "chat_collection",
			CollectionName: "dlq_jobs",
		},
	}
}

func (wp *WorkerPool) popJob(ctx context.Context) (string, error) {
	result, err := wp.atomicPop.Run(ctx, wp.Redis, []string{"priority_queue"}).Result()
	if err == redis.Nil || result == nil {
		return "", nil // no jobs available
	}
	if err != nil {
		return "", err
	}

	return result.(string), nil
}

func (wp *WorkerPool) Start(parentCtx context.Context) {
	log.Info().Msgf("Starting worker pool with %d workers", wp.WorkerNum)

	for i := 0; i < wp.WorkerNum; i++ {
		wp.wg.Add(1)
		go func(id int) {
			defer wp.wg.Done()
			wp.worker(i)
		}(i)
	}

	// start DLQ workers
	wp.wg.Add(1)
	go func() {
		defer wp.wg.Done()
		wp.StartDLQWorker(wp.ctx) // Existing: Redis DLQ -> MongoDB
	}()

	wp.wg.Add(1)
	go func() {
		defer wp.wg.Done()
		wp.StartDLQRetryConsumer(wp.ctx) // MongoDB -> Retry
	}()

	// Job producer - get jobs from redis and distribute to workers
	wp.wg.Add(1)
	go func() {
		defer wp.wg.Done()
		defer close(wp.JobChannel) // close channel when producer stop

		ticker := time.NewTicker(100 * time.Millisecond) // Poll every 100ms
		defer ticker.Stop()

		for {
			select {
			case <-wp.ctx.Done():
				log.Info().Msg("Stopping worker pool")
				return
			case <-parentCtx.Done():
				log.Info().Msg("Parent context cancelled, stopping producer")
				wp.cancel() // cancel internal context
				return
			case <-ticker.C:
				// atomic pop from redis
				payload, err := wp.popJob(wp.ctx)
				if err != nil {
					log.Error().Err(err).Msg("failed to pop job from redis")
					continue
				}

				if payload == "" {
					continue // no jobs available
				}

				// validate job before send to channel
				var job queue.Job
				if err := json.Unmarshal([]byte(payload), &job); err != nil {
					log.Error().Err(err).Msg("invalid job format, skipping")
					continue
				}

				// check job expiry
				now := time.Now().Unix()
				if now > job.ExpireAt {
					log.Warn().Str("job_id", job.ID).Int64("expired_at", job.ExpireAt).Int64("current_time", now).Msg("job expired, moving to DLQ")
					job.ErrorMsg = "Job expired"
					wp.moveToDLQ(job)
					continue
				}
				// Send to worker channel (non-blocking)
				select {
				case wp.JobChannel <- payload:
					// job sent successfully
				case <-wp.ctx.Done():
					wp.requeueJob(job)
					return
				}
			}
		}
	}()
}

func (wp *WorkerPool) worker(id int) {
	log.Info().Msgf("Worker %d started", id)
	defer log.Info().Msgf("Worker %d stopped", id)
	for {
		select {
		case <-wp.ctx.Done():
			log.Info().Msgf("Worker %d stopping", id)
			return
		case payload, ok := <-wp.JobChannel:
			if !ok {
				// channel closed, worker should stop
				return
			}

			var job queue.Job
			if err := json.Unmarshal([]byte(payload), &job); err != nil {
				log.Warn().Err(err).Msgf("Worker %d: Failed to unmarshal job payload", id)
				continue
			}
			log.Info().
				Str("job_id", job.ID).
				Str("type", job.Type).
				Msgf("Worker %d: Processing job", id)

			// process job
			if err := HandleJob(wp.ctx, job, wp.Redis, wp.ws); err != nil {
				wp.handlerJobFailure(job, err, id)
			} else {
				log.Info().
					Str("job_id", job.ID).
					Str("type", job.Type).
					Msgf("Worker %d: Job completed successfully", id)
			}
		}
	}
}

func (wp *WorkerPool) handlerJobFailure(job queue.Job, jobErr error, workerID int) {
	job.Retry++
	job.ErrorMsg = jobErr.Error()

	now := time.Now().Unix()
	if job.Retry >= job.MaxRetry || now > job.ExpireAt {
		log.Error().Str("job_id", job.ID).Int("retry", job.Retry).Int("max_retry", job.MaxRetry).Msg("Job max retries reached, moving to DLQ")

		wp.moveToDLQ(job)
		sendDLA(job) // Dead Letter Alert
	} else {
		wp.scheduleRetry(job, workerID)
	}
}

func (wp *WorkerPool) scheduleRetry(job queue.Job, workerID int) {
	// Exponential backoff: 5s, 10s, 20s, ...
	backoffSeconds := 5 * (1 << (job.Retry - 1))
	delay := time.Duration(backoffSeconds) * time.Second
	retryAt := time.Now().Add(delay).Unix()

	// consistent scoring: priority first, then retry time
	score := float64(job.Priority)*1e10 + float64(retryAt)

	jobBytes, _ := json.Marshal(job)

	if err := wp.Redis.ZAdd(wp.ctx, "priority_queue", redis.Z{
		Score:  score,
		Member: jobBytes,
	}).Err(); err != nil {
		log.Error().Err(err).Str("job_id", job.ID).Msg("failed to schedule retry, moving to DLQ")
		wp.moveToDLQ(job)
		return
	}

	log.Warn().Str("job_id", job.ID).Int("retry", job.Retry).Int("max_retry", job.MaxRetry).Float64("delay_seconds", delay.Seconds()).Msgf("Worker %d: Job scheduled for retry", workerID)
}

func (wp *WorkerPool) moveToDLQ(job queue.Job) {
	dlqBytes, _ := json.Marshal(job)
	err := wp.Redis.RPush(wp.ctx, "priority_queue_dlq", dlqBytes).Err()
	if err != nil {
		log.Error().Err(err).Str("job_id", job.ID).Msg("failed to move job to DLQ - job lost!")
	}
}

func (wp *WorkerPool) requeueJob(job queue.Job) {
	// put job back to queue with original priority
	score := float64(job.Priority)*1e10 + float64(job.CreatedAt)
	jobBytes, _ := json.Marshal(job)

	err := wp.Redis.ZAdd(wp.ctx, "priority_queue", redis.Z{
		Score:  score,
		Member: jobBytes,
	}).Err()

	if err != nil {
		log.Error().Err(err).Str("job_id", job.ID).Msg("failed to requeue job")
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

func (wp *WorkerPool) Stop() {
	log.Info().Msg("Stopping worker pool gracefully...")

	// Cancel context to signal all goroutines to stop
	wp.cancel()

	// Wait for all workers to finish current jobs
	wp.wg.Wait()

	log.Info().Msg("Worker pool stopped successfully")
}
