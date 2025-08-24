package chat_repo

import (
	"context"

	"github.com/xenn00/chat-system/internal/entity"
	app_error "github.com/xenn00/chat-system/internal/errors"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ChatRepoContract interface {
	FindOrCreateRoom(ctx context.Context, senderID, receiverID string) (*entity.Room, *app_error.AppError)
	InsertMessage(ctx context.Context, roomId, senderId string, content string) (primitive.ObjectID, *app_error.AppError)
	UpdateRoomMetadata(ctx context.Context, roomId, senderId string, msgId primitive.ObjectID) error
}
