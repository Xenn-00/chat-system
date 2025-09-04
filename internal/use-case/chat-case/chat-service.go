package chat_service

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/xenn00/chat-system/internal/dtos/chat_dto"
	"github.com/xenn00/chat-system/internal/entity"
	app_error "github.com/xenn00/chat-system/internal/errors"
	chat_repo "github.com/xenn00/chat-system/internal/repo/chat"
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
	// // return
	return &chat_dto.GetPrivateMessagesResponse{
		Messages:   respMessages,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}, nil
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
	// response with reply message dto
	return &chat_dto.ReplyPrivateMessageResponse{
		MessageID:  objID.Hex(),
		RoomID:     roomID,
		SenderID:   senderID,
		ReceiverID: msg.ReceiverID,
		Content:    msg.Content,
		ReplyTo: chat_dto.ReplyMessage{
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

	if len(roomMember) != 2 {
		return app_error.NewAppError(http.StatusBadRequest, "room must have exactly 2 members, not private", "invalid-room")
	}
	isMember := false
	for _, member := range roomMember {
		if member.UserID == receiverID {
			isMember = true
		}
	}

	if !isMember {
		return app_error.NewAppError(http.StatusForbidden, "you are not a member of this room", "forbidden")
	}

	msg, err := c.ChatRepo.FindMessageByID(ctx, messageID)
	if err != nil {
		return err
	}

	if msg.RoomID != roomID {
		return app_error.NewAppError(http.StatusBadRequest, "the message does not belong to this room", "forbidden")
	}

	if msg.ReceiverID != receiverID {
		return app_error.NewAppError(http.StatusForbidden, "you are not the receiver of this message", "forbidden")
	}
	if msg.IsRead {
		return nil
	}
	if err := c.ChatRepo.MarkMessageAsRead(ctx, messageID); err != nil {
		return err
	}
	return nil
}
