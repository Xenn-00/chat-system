package chat_service

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/xenn00/chat-system/internal/dtos/chat_dto"
	"github.com/xenn00/chat-system/internal/entity"
	app_error "github.com/xenn00/chat-system/internal/errors"
	chat_repo "github.com/xenn00/chat-system/internal/repo/chat"
	"github.com/xenn00/chat-system/internal/utils"
	"github.com/xenn00/chat-system/state"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ChatService struct {
	AppState *state.AppState
	ChatRepo chat_repo.ChatRepoContract
	// WS       *websocket.Hub
}

func NewChatService(appState *state.AppState) ChatServiceContract {
	return &ChatService{
		AppState: appState,
		ChatRepo: chat_repo.NewChatRepo(appState),
		// WS:       ws,
	}
}

const PrivateRoomMemberCount = 2

func createMessageCacheKey(roomId string) string {
	return fmt.Sprintf("chat:%s", roomId)
}

func (c *ChatService) SendPrivateMessage(ctx context.Context, req chat_dto.SendPrivateMessageRequest, senderID, receiverID string) (*chat_dto.SendPrivateMessageResponse, *app_error.AppError) {
	room, err := c.ChatRepo.FindOrCreateRoom(ctx, senderID, receiverID)
	if err != nil {
		return nil, err
	}

	msg := &entity.Message{
		ID:         primitive.NewObjectID(),
		RoomID:     room.ID.String(),
		SenderID:   senderID,
		ReceiverID: receiverID,
		Content:    req.Content,
		IsRead:     false,
		IsEdited:   false,
		CreatedAt:  time.Now(),
	}

	msgId, err := c.ChatRepo.CreateMessage(ctx, msg)
	if err != nil {
		return nil, err
	}

	if err := c.ChatRepo.UpdateRoomMetadata(ctx, room.ID.String(), senderID, msgId); err != nil {
		return nil, app_error.NewAppError(http.StatusInternalServerError, fmt.Sprintf("failed to update metadata message: %v", err), "update-room-meta")
	}

	return &chat_dto.SendPrivateMessageResponse{
		MessageID:  msgId.Hex(),
		RoomID:     room.ID.String(),
		SenderID:   senderID,
		ReceiverID: receiverID,
		Content:    req.Content,
		IsRead:     msg.IsRead,
		CreatedAt:  room.CreatedAt,
	}, nil
}

func (c *ChatService) GetPrivateMessage(ctx context.Context, req chat_dto.GetPrivateMessagesRequest, roomID string) (*chat_dto.GetPrivateMessagesResponse, *app_error.AppError) {
	// check cache
	cacheKey := createMessageCacheKey(roomID)

	cachedMessage, err := utils.GetCacheData[chat_dto.GetPrivateMessagesResponse](c.AppState.Ctx, c.AppState.Redis, cacheKey)
	if err != nil {
		log.Warn().Msgf("cache miss, '%s'", cacheKey)
	}

	if cachedMessage != nil {
		return cachedMessage, nil
	}

	// validate room exist
	room, err := c.ChatRepo.FindRoomByID(ctx, roomID)
	if err != nil {
		return nil, err
	}
	// get messages from repo (utilize cursor pagination)
	limit := req.Limit
	if limit == 0 {
		limit = 20
	}

	messages, err := c.ChatRepo.GetPrivateMessages(ctx, room.ID.String(), limit, req.BeforeID)
	if err != nil {
		return nil, err
	}
	// convert to dto
	respMessages := make([]chat_dto.PrivateMessages, 0, len(messages))
	for _, msg := range messages {
		var replyTo *chat_dto.ReplyMessage
		if msg.ReplyTo != nil {
			replyTo = &chat_dto.ReplyMessage{
				RepliedMessageID: msg.ReplyTo.MessageID.Hex(),
				Content:          msg.ReplyTo.Content,
				SenderID:         msg.ReplyTo.SenderID,
			}
		}
		respMessages = append(respMessages, chat_dto.PrivateMessages{
			MessageID:  msg.ID.Hex(),
			RoomID:     msg.RoomID,
			SenderID:   msg.SenderID,
			ReceiverID: msg.ReceiverID,
			Content:    msg.Content,
			ReplyTo:    replyTo,
			IsRead:     msg.IsRead,
			CreatedAt:  msg.CreatedAt,
		})
	}
	// // determine next cursor and has more
	var nextCursor *string
	if len(messages) > 0 {
		lastMsgID := messages[len(messages)-1].ID.Hex()
		nextCursor = &lastMsgID
	}

	hasMore := len(messages) == req.Limit

	res := &chat_dto.GetPrivateMessagesResponse{
		Messages:   respMessages,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}
	// set cache
	utils.SetCacheData(c.AppState.Ctx, c.AppState.Redis, cacheKey, res, time.Minute*5)
	// // return
	return res, nil
}

func (c *ChatService) ReplyPrivateMessage(ctx context.Context, req chat_dto.ReplyPrivateMessageRequest, senderID, roomID string) (*chat_dto.ReplyPrivateMessageResponse, *app_error.AppError) {
	// validate room exist
	if _, err := c.ChatRepo.FindRoomByID(ctx, roomID); err != nil {
		return nil, err
	}
	// validate room member (sender and receiver is member of the room)
	members, err := c.ChatRepo.FindRoomMembers(ctx, roomID)
	if err != nil {
		return nil, err
	}
	if len(members) != 2 {
		return nil, app_error.NewAppError(http.StatusBadRequest, "room must have exactly 2 members, not private", "invalid-room")
	}
	isMember := false
	for _, member := range members {
		if member.UserID == senderID || member.UserID == req.ReceiverID {
			isMember = true
		}
	}
	if !isMember {
		return nil, app_error.NewAppError(http.StatusForbidden, "you are not a member of this room", "forbidden")
	}
	// validate reply_to message exist in the room
	repliedMsg, err := c.ChatRepo.FindMessageByID(ctx, req.ReplyTo)
	if err != nil {
		return nil, err
	}

	if repliedMsg.RoomID != roomID {
		return nil, app_error.NewAppError(http.StatusBadRequest, "the message you are replying to does not belong to this room", "forbidden")
	}

	msg := &entity.Message{
		ID:         primitive.NewObjectID(),
		RoomID:     roomID,
		SenderID:   senderID,
		ReceiverID: req.ReceiverID,
		Content:    req.Content,
		ReplyTo: &entity.ReplyTo{
			MessageID: repliedMsg.ID,
			Content:   repliedMsg.Content,
			SenderID:  repliedMsg.SenderID,
		},
		IsRead:    false,
		IsEdited:  false,
		CreatedAt: time.Now(),
	}

	objID, err := c.ChatRepo.ReplyMessage(ctx, msg)
	if err != nil {
		return nil, err
	}

	// invalidate cache key
	cacheKey := createMessageCacheKey(roomID)
	utils.DeleteCacheData(c.AppState.Ctx, c.AppState.Redis, cacheKey)

	// response with reply message dto
	return &chat_dto.ReplyPrivateMessageResponse{
		MessageID:  objID.Hex(),
		RoomID:     roomID,
		SenderID:   senderID,
		ReceiverID: msg.ReceiverID,
		Content:    msg.Content,
		ReplyTo: &chat_dto.ReplyMessage{
			RepliedMessageID: repliedMsg.ID.Hex(),
			Content:          repliedMsg.Content,
			SenderID:         repliedMsg.SenderID,
		},
		IsRead:    msg.IsRead,
		CreatedAt: msg.CreatedAt,
	}, nil
}

func (c *ChatService) MarkPrivateMessageAsRead(ctx context.Context, receiverID, roomID, messageID string) *app_error.AppError {
	// validate room, message, and receiver is member of the room
	roomMember, err := c.ChatRepo.FindRoomMembers(ctx, roomID)
	if err != nil {
		return err
	}

	if len(roomMember) != PrivateRoomMemberCount {
		return app_error.NewAppError(http.StatusBadRequest, "room must have exactly 2 members, not private", "invalid-room")
	}

	if !c.isUserMemberOfRoom(roomMember, receiverID) {
		return app_error.NewAppError(http.StatusForbidden, "you are not a member of this room", "forbidden")
	}

	msg, err := c.ChatRepo.FindMessageByID(ctx, messageID)
	if err != nil {
		return err
	}

	if msg.RoomID != roomID {
		return app_error.NewAppError(http.StatusBadRequest, "the message does not belong to this room", "forbidden")
	}

	if msg.SenderID == receiverID {
		return app_error.NewAppError(http.StatusBadRequest, "cannot mark your own message as read", "invalid-action")
	}

	if msg.IsRead {
		return nil
	}

	// invalidate cache key
	cacheKey := createMessageCacheKey(roomID)
	utils.DeleteCacheData(c.AppState.Ctx, c.AppState.Redis, cacheKey)

	return c.ChatRepo.MarkMessageAsRead(ctx, messageID)
}

func (c *ChatService) UpdatePrivateMessage(ctx context.Context, req chat_dto.UpdatePrivateMessageRequest, senderID, roomID, messageID string) (*chat_dto.UpdatePrivateMessageResponse, *app_error.AppError) {
	// get original message
	originalMsg, err := c.ChatRepo.FindMessageByID(ctx, messageID)
	if err != nil {
		return nil, err
	}
	log.Info().Msgf("room id: %s", originalMsg.RoomID)
	// authorization check
	if originalMsg.SenderID != senderID {
		return nil, app_error.NewAppError(http.StatusForbidden, "You can only update your own message", "authorization")
	}
	if originalMsg.RoomID != roomID {
		return nil, app_error.NewAppError(http.StatusForbidden, "You are not a member of this chat room", "authorization")
	}
	// Time window check
	editWindow := 15 * time.Minute
	if time.Since(originalMsg.CreatedAt) > editWindow {
		return nil, app_error.NewAppError(http.StatusForbidden, "Message edit time window expired", "time_expired")
	}
	// room & membership validation
	room, err := c.ChatRepo.FindRoomByID(ctx, originalMsg.RoomID)
	log.Info().Msgf("room deletedAt: %v", room.DeletedAt)
	log.Info().Msgf("room err: %v", err)
	if err != nil || room.DeletedAt != nil {
		return nil, app_error.NewAppError(http.StatusNotFound, "Room not found or inactive", "room")
	}
	member, err := c.ChatRepo.FindRoomMembers(ctx, originalMsg.RoomID)
	log.Info().Msgf("is member: %v", c.isUserMemberOfRoom(member, senderID))
	if err != nil || !c.isUserMemberOfRoom(member, senderID) {
		log.Error().Err(err).Msgf("an error occur: %v", err)
		return nil, app_error.NewAppError(http.StatusForbidden, fmt.Sprintf("%s, you are not a member of this room", senderID), "forbidden")
	}
	// Content validation (no empty, different from original)
	if strings.TrimSpace(req.Content) == strings.TrimSpace(originalMsg.Content) {
		return nil, app_error.NewAppError(http.StatusBadRequest, "New content must be different", "content")
	}
	// update message with optimistic locking
	now := time.Now()
	updatedMsg := &entity.Message{
		ID:        originalMsg.ID,
		Content:   req.Content,
		IsEdited:  true,
		UpdatedAt: &now,
	}

	messageEdit := &entity.MessageEditEntry{
		MessageID:       originalMsg.ID,
		OriginalContent: originalMsg.Content,
		NewContent:      req.Content,
		EditedBy:        originalMsg.SenderID,
		EditedAt:        now,
	}

	err = c.ChatRepo.UpdateMessage(ctx, updatedMsg, messageEdit, originalMsg.UpdatedAt)
	if err != nil {
		return nil, err
	}

	// invalidate cache key
	cacheKey := createMessageCacheKey(roomID)
	utils.DeleteCacheData(c.AppState.Ctx, c.AppState.Redis, cacheKey)

	messageHistory := make([]*chat_dto.MessageEditEntry, 0)

	if len(originalMsg.MessageEditHistory) == 0 {
		messageHistory = append(messageHistory, &chat_dto.MessageEditEntry{
			MessageID:       messageEdit.MessageID.Hex(),
			OriginalContent: messageEdit.OriginalContent,
			NewContent:      messageEdit.NewContent,
			EditedBy:        messageEdit.EditedBy,
			EditedAt:        messageEdit.EditedAt,
		})
	} else {
		for _, entry := range originalMsg.MessageEditHistory {
			messageHistory = append(messageHistory, &chat_dto.MessageEditEntry{
				MessageID:       entry.MessageID.Hex(),
				OriginalContent: entry.OriginalContent,
				NewContent:      entry.NewContent,
				EditedBy:        entry.EditedBy,
				EditedAt:        entry.EditedAt,
			})
		}
	}

	var replyTo *chat_dto.ReplyMessage
	if originalMsg.ReplyTo != nil {
		replyTo = &chat_dto.ReplyMessage{
			RepliedMessageID: originalMsg.ReplyTo.MessageID.Hex(),
			Content:          originalMsg.ReplyTo.Content,
			SenderID:         originalMsg.ReplyTo.SenderID,
		}
	}

	return &chat_dto.UpdatePrivateMessageResponse{
		MessageID:          originalMsg.ID.Hex(),
		RoomID:             originalMsg.RoomID,
		SenderID:           originalMsg.SenderID,
		ReceiverID:         originalMsg.ReceiverID,
		Content:            updatedMsg.Content,
		MessageEditHistory: messageHistory,
		ReplyTo:            replyTo,
		IsRead:             originalMsg.IsRead,
		IsEdited:           updatedMsg.IsEdited,
		UpdatedAt:          *updatedMsg.UpdatedAt,
	}, nil
}

func (c *ChatService) isUserMemberOfRoom(members []*entity.RoomMember, userID string) bool {
	for _, member := range members {
		log.Info().Msgf("member_id: %v", member.UserID)
		if member.UserID == userID {
			return member.LeftAt == nil
		}
	}

	return false
}
