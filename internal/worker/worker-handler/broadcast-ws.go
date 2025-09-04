package worker_handler

import (
	"encoding/json"
	"fmt"

	"github.com/xenn00/chat-system/internal/utils/types"
	"github.com/xenn00/chat-system/internal/websocket"
)

func (wh *WorkerHandler) HandleBroadcastPrivateMessage(raw json.RawMessage) error {
	var payload types.BroadcastMessagePayload

	if err := json.Unmarshal(raw, &payload); err != nil {
		return fmt.Errorf("invalid broadcast payload: %w", err)
	}

	msg := websocket.Message{
		Type:      "chat_message",
		RoomId:    payload.RoomID,
		SenderID:  payload.SenderID,
		Content:   payload.Content,
		Timestamp: payload.CreatedAt.Unix(),
	}

	wh.Ws.BroadcastToRoom(payload.RoomID, msg)

	return nil
}

func (wh *WorkerHandler) HandleBroadcasPrivateMessageReply(raw json.RawMessage) error {
	var payload types.BroadcastMessagePayload

	if err := json.Unmarshal(raw, &payload); err != nil {
		return fmt.Errorf("invalid broadcast payload: %w", err)
	}
	msg := websocket.Message{
		Type:      "chat_message",
		RoomId:    payload.RoomID,
		SenderID:  payload.SenderID,
		Content:   payload.Content,
		Timestamp: payload.CreatedAt.Unix(),
		Reply: &websocket.ReplyMessage{
			MessageID: payload.ReplyTo.MessageID,
			Content:   payload.ReplyTo.Content,
			SenderID:  payload.ReplyTo.SenderID,
		},
	}

	wh.Ws.BroadcastToRoom(payload.RoomID, msg)
	return nil
}
