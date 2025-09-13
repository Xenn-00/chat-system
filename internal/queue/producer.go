package queue

import (
	"context"
	"encoding/json"

	"github.com/redis/go-redis/v9"
)

type Producer interface {
	Enqueue(ctx context.Context, job Job) error
}

type RedisProducer struct {
	Redis *redis.Client
}

func NewProducer(redis *redis.Client) Producer {
	return &RedisProducer{Redis: redis}
}

func (p *RedisProducer) Enqueue(ctx context.Context, job Job) error {
	jobBytes, err := json.Marshal(job)
	if err != nil {
		return err
	}

	// score = priority * 1e10 + ExpireAt
	score := float64(job.Priority)*1e10 + float64(job.ExpireAt)
	return p.Redis.ZAdd(ctx, "priority_queue", redis.Z{
		Score:  score,
		Member: jobBytes,
	}).Err()
}
