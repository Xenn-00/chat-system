package chat_service

import (
	"context"
	"fmt"
	"net/http"

	"github.com/xenn00/chat-system/internal/dtos/chat_dto"
	app_error "github.com/xenn00/chat-system/internal/errors"
	chat_repo "github.com/xenn00/chat-system/internal/repo/chat"
	"github.com/xenn00/chat-system/state"
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

	msgId, err := c.ChatRepo.InsertMessage(ctx, room.ID.String(), senderID, receiverID, req.Content)
	if err != nil {
		return nil, err
	}

	if err := c.ChatRepo.UpdateRoomMetadata(ctx, room.ID.String(), senderID, msgId); err != nil {
		return nil, app_error.NewAppError(http.StatusInternalServerError, fmt.Sprintf("failed to update metadata message: %v", err), "update-room-meta")
	}

	return &chat_dto.SendPrivateMessageResponse{
		MessageID: msgId.String(),
		RoomID:    room.ID.String(),
		SenderID:  senderID,
		Content:   req.Content,
		CreatedAt: room.CreatedAt,
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
		respMessages = append(respMessages, chat_dto.PrivateMessages{
			MessageID:  msg.ID.Hex(),
			SenderID:   msg.SenderID,
			ReceiverID: msg.ReceiverID,
			Content:    msg.Content,
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
