package chat_repo

import (
	"context"
	"time"

	"github.com/xenn00/chat-system/internal/entity"
	app_error "github.com/xenn00/chat-system/internal/errors"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ChatRepoContract interface {
	FindOrCreateRoom(ctx context.Context, senderID, receiverID string) (*entity.Room, *app_error.AppError)
	FindRoomByID(ctx context.Context, roomID string) (*entity.Room, *app_error.AppError)
	FindRoomMembers(ctx context.Context, roomID string) ([]*entity.RoomMember, *app_error.AppError)
	CreateMessage(ctx context.Context, msg *entity.Message) (primitive.ObjectID, *app_error.AppError)
	ReplyMessage(ctx context.Context, msg *entity.Message) (primitive.ObjectID, *app_error.AppError)
	UpdateRoomMetadata(ctx context.Context, roomID, senderID string, msgId primitive.ObjectID) error
	GetPrivateMessages(ctx context.Context, roomID string, limit int, beforeID *string) ([]*entity.Message, *app_error.AppError)
	FindMessageByID(ctx context.Context, messageID string) (*entity.Message, *app_error.AppError)
	MarkMessageAsRead(ctx context.Context, messageID string) *app_error.AppError
	UpdateMessage(ctx context.Context, msg *entity.Message, messageEditEntry *entity.MessageEditEntry, originalTimestamp *time.Time) *app_error.AppError
}
