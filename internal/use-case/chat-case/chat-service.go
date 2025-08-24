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

func (c *ChatService) CreateRoomChat(ctx context.Context, req chat_dto.CreateRoomRequest) (*chat_dto.CreateRoomResponse, *app_error.AppError) {
	return nil, nil
}

func (c *ChatService) SendPrivateMessage(ctx context.Context, req chat_dto.SendPrivateMessageRequest, senderID string) (*chat_dto.SendPrivateMessageResponse, *app_error.AppError) {
	room, err := c.ChatRepo.FindOrCreateRoom(ctx, senderID, req.ReceiverID)
	if err != nil {
		return nil, err
	}

	msgId, err := c.ChatRepo.InsertMessage(ctx, room.ID.String(), senderID, req.Content)
	if err != nil {
		return nil, err
	}

	if err := c.ChatRepo.UpdateRoomMetadata(ctx, room.ID.String(), senderID, msgId); err != nil {
		return nil, app_error.NewAppError(http.StatusInternalServerError, fmt.Sprintf("failed to update metadata message: %v", err), "update-room-meta")
	}

	// c.WS.BroadcastToRoom(room.ID.String(), websocket.Message{
	// 	Type:      "chat_message",
	// 	RoomId:    room.ID.String(),
	// 	SenderID:  senderID,
	// 	Content:   req.Content,
	// 	Timestamp: time.Now().Unix(),
	// })

	return &chat_dto.SendPrivateMessageResponse{
		MessageID: msgId.String(),
		RoomID:    room.ID.String(),
		SenderID:  senderID,
		Content:   req.Content,
		CreatedAt: room.CreatedAt,
	}, nil
}
