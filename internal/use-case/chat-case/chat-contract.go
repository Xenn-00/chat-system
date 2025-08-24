package chat_service

import (
	"context"

	"github.com/xenn00/chat-system/internal/dtos/chat_dto"
	app_error "github.com/xenn00/chat-system/internal/errors"
)

type ChatServiceContract interface {
	CreateRoomChat(ctx context.Context, req chat_dto.CreateRoomRequest) (*chat_dto.CreateRoomResponse, *app_error.AppError)
	SendPrivateMessage(ctx context.Context, req chat_dto.SendPrivateMessageRequest, senderID string) (*chat_dto.SendPrivateMessageResponse, *app_error.AppError)
}
