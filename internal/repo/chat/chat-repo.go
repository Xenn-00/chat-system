package chat_repo

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/xenn00/chat-system/internal/entity"
	app_error "github.com/xenn00/chat-system/internal/errors"
	"github.com/xenn00/chat-system/state"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"gorm.io/gorm"
)

type ChatRepo struct {
	AppState *state.AppState
}

func NewChatRepo(appState *state.AppState) ChatRepoContract {
	return &ChatRepo{
		AppState: appState,
	}
}

func (r *ChatRepo) FindOrCreateRoom(ctx context.Context, senderID, receiverID string) (*entity.Room, *app_error.AppError) {
	var room entity.Room

	tx := r.AppState.DB.WithContext(ctx).Begin()

	err := tx.WithContext(ctx).Joins("JOIN room_members m1 ON rooms.id = m1.room_id").Joins("JOIN room_members m2 ON rooms.id = m2.room_id").Where("rooms.rt = ?", "private").Where("m1.user_id = ?", senderID).Where("m2.user_id = ?", receiverID).First(&room).Error

	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		tx.Rollback()
		return nil, app_error.NewAppError(http.StatusInternalServerError, "unexpected error occur when fetch chat room", "db-error")
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		newRoom := &entity.Room{
			ID:        uuid.New(),
			RT:        "private",
			CreatedBy: senderID,
		}

		if err := tx.Create(newRoom).Error; err != nil {
			tx.Rollback()
			return nil, app_error.NewAppError(http.StatusInternalServerError, "Failed to created private room", "db-error")
		}

		members := &[]entity.RoomMember{
			{
				RoomID: newRoom.ID.String(),
				UserID: senderID,
			},
			{
				RoomID: newRoom.ID.String(),
				UserID: receiverID,
			},
		}

		if err := tx.Create(members).Error; err != nil {
			tx.Rollback()
			return nil, app_error.NewAppError(http.StatusInternalServerError, "Failed to add member to private room", "db-error")
		}

		if err := tx.Commit().Error; err != nil {
			return nil, app_error.NewAppError(http.StatusInternalServerError, "unexpected error occur when commit create room private chat", "db-error")
		}

		return newRoom, nil
	}

	if err := tx.Commit().Error; err != nil {
		return nil, app_error.NewAppError(http.StatusInternalServerError, "already has private chat room", "tx")
	}

	return &room, nil
}

func (r *ChatRepo) InsertMessage(ctx context.Context, roomId, senderId string, content string) (primitive.ObjectID, *app_error.AppError) {
	coll := r.AppState.Mongo.Database("chatdb").Collection("messages")

	msg := entity.Message{
		RoomID:   roomId,
		SenderID: senderId,
		Content:  content,
		IsRead:   false,
		CreateAt: time.Now(),
	}

	res, err := coll.InsertOne(ctx, msg)
	if err != nil {
		return primitive.NilObjectID, app_error.NewAppError(http.StatusInternalServerError, fmt.Sprintf("failed to add msg: %v", err), "mongo")
	}

	oid, ok := res.InsertedID.(primitive.ObjectID)
	if !ok {
		return primitive.NilObjectID, app_error.NewAppError(http.StatusInternalServerError, fmt.Sprintf("failed to cast inserted ID to ObjectID: %v", err), "casting-objectID")
	}

	return oid, nil
}

func (r *ChatRepo) UpdateRoomMetadata(ctx context.Context, roomId, senderId string, msgId primitive.ObjectID) error {
	tx := r.AppState.DB.WithContext(ctx).Begin()

	if err := tx.Model(&entity.RoomMember{}).Where("room_id = ? AND user_id = ?", roomId, senderId).Updates(map[string]any{
		"last_message_id": msgId.Hex(),
		"last_message_at": time.Now(),
		"unread_count":    gorm.Expr("unread_count + ?", 1),
	}).Error; err != nil {
		tx.Rollback()
		return app_error.NewAppError(http.StatusInternalServerError, "failed to update last message metadata", "db-error")
	}

	return tx.Commit().Error
}
