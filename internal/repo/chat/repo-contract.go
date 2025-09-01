package chat_repo

import (
	"context"

	"github.com/xenn00/chat-system/internal/entity"
	app_error "github.com/xenn00/chat-system/internal/errors"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ChatRepoContract interface {
	FindOrCreateRoom(ctx context.Context, senderID, receiverID string) (*entity.Room, *app_error.AppError)
	FindRoomByID(ctx context.Context, roomID string) (*entity.Room, *app_error.AppError)
	InsertMessage(ctx context.Context, roomID, senderID, receiverID string, content string) (primitive.ObjectID, *app_error.AppError)
	UpdateRoomMetadata(ctx context.Context, roomID, senderID string, msgId primitive.ObjectID) error
	GetPrivateMessages(ctx context.Context, roomID string, limit int, beforeID *string) ([]*entity.Message, *app_error.AppError)
}
