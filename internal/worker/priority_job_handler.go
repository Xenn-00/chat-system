package worker

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/redis/go-redis/v9"
	"github.com/xenn00/chat-system/internal/queue"
	worker_handler "github.com/xenn00/chat-system/internal/worker/worker-handler"
)

type JobPayload struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

func HandleJob(ctx context.Context, job queue.Job, redis *redis.Client) error {
	switch job.Type {
	case "create_user_otp":
		return worker_handler.HandlerCreateUserOTP(ctx, redis, job.Payload)
	default:
		return fmt.Errorf("unknown job type: %s", job.Type)
	}
}
