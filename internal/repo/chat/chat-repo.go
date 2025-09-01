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
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
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
				Role:   "member",
			},
			{
				RoomID: newRoom.ID.String(),
				UserID: receiverID,
				Role:   "member",
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

func (r *ChatRepo) FindRoomByID(ctx context.Context, roomID string) (*entity.Room, *app_error.AppError) {
	var room entity.Room
	if err := r.AppState.DB.WithContext(ctx).Where("id = ?", roomID).First(&room).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, app_error.NewAppError(http.StatusNotFound, "room not found", "not-found")
		}
		return nil, app_error.NewAppError(http.StatusInternalServerError, "failed to fetch room", "db-error")
	}
	return &room, nil
}

func (r *ChatRepo) InsertMessage(ctx context.Context, roomID, senderID, receiverID string, content string) (primitive.ObjectID, *app_error.AppError) {
	coll := r.AppState.Mongo.Database("chat_collection").Collection("messages")

	msg := entity.Message{
		ID:         primitive.NewObjectID(),
		RoomID:     roomID,
		SenderID:   senderID,
		ReceiverID: receiverID,
		Content:    content,
		IsRead:     false,
		CreatedAt:  time.Now(),
	}

	_, err := coll.InsertOne(ctx, msg)
	if err != nil {
		return primitive.NilObjectID, app_error.NewAppError(http.StatusInternalServerError, fmt.Sprintf("failed to add msg: %v", err), "mongo")
	}

	return msg.ID, nil
}

func (r *ChatRepo) UpdateRoomMetadata(ctx context.Context, roomID, senderID string, msgId primitive.ObjectID) error {
	tx := r.AppState.DB.WithContext(ctx).Begin()

	if err := tx.Model(&entity.RoomMember{}).Where("room_id = ? AND user_id = ?", roomID, senderID).Updates(map[string]any{
		"last_read_msg_id": msgId.Hex(),
		"last_message_at":  time.Now(),
		"unread_count":     gorm.Expr("unread_count + ?", 1),
	}).Error; err != nil {
		tx.Rollback()
		return app_error.NewAppError(http.StatusInternalServerError, "failed to update last message metadata", "db-error")
	}

	return tx.Commit().Error
}

func (r *ChatRepo) GetPrivateMessages(ctx context.Context, roomID string, limit int, beforeID *string) ([]*entity.Message, *app_error.AppError) {
	collection := r.AppState.Mongo.Database("chat_collection").Collection("messages")

	// base filter: all messages in the room
	filter := bson.M{"room_id": roomID}

	// if beforeID is provided -> filter messages with ID < beforeID
	if beforeID != nil {
		objID, err := primitive.ObjectIDFromHex(*beforeID)
		if err != nil {
			return nil, app_error.NewAppError(http.StatusBadRequest, fmt.Sprintf("error when trying to parse before_id: %v", err), "before-id")
		}
		filter["_id"] = bson.M{"$lt": objID}
	}

	cur, err := collection.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "_id", Value: -1}}).SetLimit(int64(limit))) // sort by _id desc to get latest messages

	if err != nil {
		return nil, app_error.NewAppError(http.StatusInternalServerError, fmt.Sprintf("failed to fetch messages: %v", err), "mongo")
	}

	defer cur.Close(ctx)

	var messages []*entity.Message

	if err := cur.All(ctx, &messages); err != nil {
		return nil, app_error.NewAppError(http.StatusInternalServerError, fmt.Sprintf("failed to decode messages: %v", err), "mongo")
	}

	// reverse messages to be in ascending order (oldest to newest)
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, nil
}
