package worker_handler

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/xenn00/chat-system/internal/utils/types"
	"github.com/xenn00/chat-system/internal/websocket"
)

func (wh *WorkerHandler) HandleBroadcastPrivateMessage(raw json.RawMessage) error {
	var payload types.BroadcastMessagePayload

	if err := json.Unmarshal(raw, &payload); err != nil {
		return fmt.Errorf("invalid broadcast payload: %w", err)
	}

	// Create chat message using the new structure
	chatData := websocket.ChatMessage{
		Type:      websocket.MessageTypeChatMessage,
		RoomID:    payload.RoomID,
		MessageID: payload.MessageID,
		SenderID:  payload.SenderID,
		Content:   payload.Content,
		IsEdited:  false,
		IsRead:    false,
		CreatedAt: payload.CreatedAt.Unix(),
		Timestamp: payload.CreatedAt.Unix(),
	}

	// Create OutgoingMessage
	msg := websocket.OutgoingMessage{
		Type:      websocket.MessageTypeChatMessage,
		RoomID:    payload.RoomID,
		MessageID: payload.MessageID,
		SenderID:  payload.SenderID,
		Data:      chatData,
		Timestamp: payload.CreatedAt.Unix(),
	}

	wh.Ws.BroadcastToRoom(payload.RoomID, msg)

	return nil
}

func (wh *WorkerHandler) HandleBroadcastPrivateMessageReply(raw json.RawMessage) error {
	var payload types.BroadcastMessagePayload

	if err := json.Unmarshal(raw, &payload); err != nil {
		return fmt.Errorf("invalid broadcast payload: %w", err)
	}

	// Create reply data
	var replyData *websocket.ReplyMessage
	if payload.ReplyTo != nil {
		replyData = &websocket.ReplyMessage{
			MessageID: payload.ReplyTo.MessageID,
			Content:   payload.ReplyTo.Content,
			SenderID:  payload.ReplyTo.SenderID,
		}
	}

	// Create chat message with reply
	chatData := websocket.ChatMessage{
		Type:      websocket.MessageTypeChatMessage,
		RoomID:    payload.RoomID,
		MessageID: payload.MessageID,
		SenderID:  payload.SenderID,
		Content:   payload.Content,
		IsEdited:  false,
		IsRead:    false,
		Reply:     replyData,
		CreatedAt: payload.CreatedAt.Unix(),
		Timestamp: payload.CreatedAt.Unix(),
	}

	// Create OutgoingMessage
	msg := websocket.OutgoingMessage{
		Type:      websocket.MessageTypeChatMessage,
		RoomID:    payload.RoomID,
		MessageID: payload.MessageID,
		SenderID:  payload.SenderID,
		Data:      chatData,
		Timestamp: payload.CreatedAt.Unix(),
	}

	wh.Ws.BroadcastToRoom(payload.RoomID, msg)
	return nil
}

func (wh *WorkerHandler) HandleBroadcastPrivateMessageUpdate(raw json.RawMessage) error {
	var payload types.BroadcastMessagePayload

	if err := json.Unmarshal(raw, &payload); err != nil {
		return fmt.Errorf("invalid broadcast payload: %w", err)
	}

	// Convert edit history to new format
	var editHistory []websocket.MessageEditEntry
	if len(payload.MessageEditHistory) > 0 {
		editHistory = make([]websocket.MessageEditEntry, len(payload.MessageEditHistory))
		for i, edit := range payload.MessageEditHistory {
			editHistory[i] = websocket.MessageEditEntry{
				MessageID:       payload.MessageID,
				OriginalContent: edit.OriginalContent,
				NewContent:      edit.NewContent,
				EditedBy:        payload.SenderID,
				EditedAt:        payload.UpdatedAt.Unix(),
			}
		}
	} else {
		log.Warn().
			Str("message_id", payload.MessageID).
			Str("room_id", payload.RoomID).
			Msg("Message edit history is empty, broadcasting without edit history")
		editHistory = []websocket.MessageEditEntry{}
	}

	updatedAt := payload.UpdatedAt.Unix()

	// Create message updated data
	updateData := websocket.MessageUpdated{
		Type:               websocket.MessageTypeMessageUpdated,
		RoomID:             payload.RoomID,
		MessageID:          payload.MessageID,
		Content:            payload.Content,
		IsEdited:           true,
		MessageEditHistory: editHistory,
		UpdatedAt:          updatedAt,
		EditedBy:           payload.SenderID,
		Timestamp:          time.Now().Unix(),
	}

	// Create OutgoingMessage
	msg := websocket.OutgoingMessage{
		Type:      websocket.MessageTypeMessageUpdated,
		RoomID:    payload.RoomID,
		MessageID: payload.MessageID,
		SenderID:  payload.SenderID,
		Data:      updateData,
		Timestamp: time.Now().Unix(),
	}

	wh.Ws.BroadcastToRoom(payload.RoomID, msg)
	return nil
}

// Alternative simplified approach using helper functions from message.go
func (wh *WorkerHandler) HandleBroadcastPrivateMessageSimplified(raw json.RawMessage) error {
	var payload types.BroadcastMessagePayload

	if err := json.Unmarshal(raw, &payload); err != nil {
		return fmt.Errorf("invalid broadcast payload: %w", err)
	}

	// Use the helper function from message.go directly
	msg := websocket.NewChatMessage(payload.RoomID, payload.MessageID, payload.SenderID, payload.Content)

	wh.Ws.BroadcastToRoom(payload.RoomID, msg)
	return nil
}

func (wh *WorkerHandler) HandleBroadcastPrivateMessageUpdateSimplified(raw json.RawMessage) error {
	var payload types.BroadcastMessagePayload

	if err := json.Unmarshal(raw, &payload); err != nil {
		return fmt.Errorf("invalid broadcast payload: %w", err)
	}

	// Convert edit history
	var editHistory []websocket.MessageEditEntry
	if len(payload.MessageEditHistory) > 0 {
		editHistory = make([]websocket.MessageEditEntry, len(payload.MessageEditHistory))
		for i, edit := range payload.MessageEditHistory {
			editHistory[i] = websocket.MessageEditEntry{
				MessageID:       payload.MessageID,
				OriginalContent: edit.OriginalContent,
				NewContent:      edit.NewContent,
				EditedBy:        payload.SenderID,
				EditedAt:        payload.UpdatedAt.Unix(),
			}
		}
	}

	// Use helper function from message.go directly
	msg := websocket.NewMessageUpdated(payload.RoomID, payload.MessageID, payload.Content, payload.SenderID, editHistory)

	wh.Ws.BroadcastToRoom(payload.RoomID, msg)
	return nil
}
