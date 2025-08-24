package worker_handler

import (
	"context"

	"github.com/redis/go-redis/v9"
	"github.com/xenn00/chat-system/internal/websocket"
)

type WorkerHandler struct {
	Ctx   context.Context
	Redis *redis.Client
	Ws    *websocket.Hub
}

func NewWorkerHandler(ctx context.Context, redis *redis.Client, ws *websocket.Hub) *WorkerHandler {
	return &WorkerHandler{
		Ctx:   ctx,
		Redis: redis,
		Ws:    ws,
	}
}
