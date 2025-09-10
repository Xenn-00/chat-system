package worker

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/redis/go-redis/v9"
	"github.com/xenn00/chat-system/internal/queue"
	"github.com/xenn00/chat-system/internal/websocket"
	worker_handler "github.com/xenn00/chat-system/internal/worker/worker-handler"
)

type JobPayload struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

func HandleJob(ctx context.Context, job queue.Job, redis *redis.Client, ws *websocket.Hub) error {
	workerHandler := worker_handler.NewWorkerHandler(ctx, redis, ws)
	switch job.Type {
	case "create_user_otp":
		return workerHandler.HandlerCreateUserOTP(ctx, redis, job.Payload)
	case "broadcast_private_message":
		return workerHandler.HandleBroadcastPrivateMessage(job.Payload)
	case "broadcast_private_message_reply":
		return workerHandler.HandleBroadcasPrivateMessageReply(job.Payload)
	case "broadcast_private_message_updated":
		return workerHandler.HandleBroadcastPrivateMessageUpdate(job.Payload)
	default:
		return fmt.Errorf("unknown job type: %s", job.Type)
	}
}
