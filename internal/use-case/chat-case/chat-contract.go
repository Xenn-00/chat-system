package chat_service

import (
	"context"

	"github.com/xenn00/chat-system/internal/dtos/chat_dto"
	app_error "github.com/xenn00/chat-system/internal/errors"
)

type ChatServiceContract interface {
	SendPrivateMessage(ctx context.Context, req chat_dto.SendPrivateMessageRequest, senderID, receiverID string) (*chat_dto.SendPrivateMessageResponse, *app_error.AppError)
	GetPrivateMessage(ctx context.Context, req chat_dto.GetPrivateMessagesRequest, roomID string) (*chat_dto.GetPrivateMessagesResponse, *app_error.AppError)
	ReplyPrivateMessage(ctx context.Context, req chat_dto.ReplyPrivateMessageRequest, senderID, roomID string) (*chat_dto.ReplyPrivateMessageResponse, *app_error.AppError)
	MarkPrivateMessageAsRead(ctx context.Context, receiverID, roomID, messageID string) *app_error.AppError
}
