package websocket

import "time"

// OutgoingMessage represents messages sent from server to client
type OutgoingMessage struct {
	Type      string      `json:"type"`
	RoomID    string      `json:"room_id,omitempty"`
	SenderID  string      `json:"sender_id,omitempty"`
	MessageID string      `json:"message_id,omitempty"`
	Data      interface{} `json:"data,omitempty"`
	Timestamp int64       `json:"timestamp"`
}

// Specific message types for better type safety

// ChatMessage represents a chat message
type ChatMessage struct {
	Type               string              `json:"type"`
	RoomID             string              `json:"room_id"`
	MessageID          string              `json:"message_id"`
	SenderID           string              `json:"senderId"`
	ReceiverID         string              `json:"receiver_id"`
	Content            string              `json:"content"`
	IsEdited           bool                `json:"is_edited"`
	IsRead             bool                `json:"is_read"`
	MessageEditHistory []MessageEditEntry  `json:"message_edit_history,omitempty"`
	Reply              *ReplyMessage       `json:"reply,omitempty"`
	Attachments        []MessageAttachment `json:"attachments,omitempty"`
	CreatedAt          int64               `json:"created_at"`
	UpdatedAt          *int64              `json:"updated_at"`
	Timestamp          int64               `json:"timestamp"`
}

// MessageUpdated represents an edited message
type MessageUpdated struct {
	Type               string             `json:"type"`
	RoomID             string             `json:"room_id"`
	MessageID          string             `json:"message_id"`
	Content            string             `json:"content"`
	IsEdited           bool               `json:"is_edited"`
	MessageEditHistory []MessageEditEntry `json:"message_edit_history,omitempty"`
	UpdatedAt          int64              `json:"updated_at"`
	EditedBy           string             `json:"edited_by"`
	Timestamp          int64              `json:"timestamp"`
}

// MessageRead represents a message read receipt
type MessageRead struct {
	Type      string `json:"type"`
	RoomID    string `json:"room_id"`
	MessageID string `json:"message_id"`
	ReadBy    string `json:"read_by"`
	ReadAt    int64  `json:"read_at"`
	Timestamp int64  `json:"timestamp"`
}

// UserTyping represents typing indicators
type UserTyping struct {
	Type      string `json:"type"`
	RoomID    string `json:"room_id"`
	UserID    string `json:"user_id"`
	IsTyping  bool   `json:"is_typing"`
	Timestamp int64  `json:"timestamp"`
}

// UserStatus represents online/offline status
type UserStatus struct {
	Type      string `json:"type"`
	RoomID    string `json:"room_id,omitempty"`
	UserID    string `json:"user_id"`
	Status    string `json:"status"`
	LastSeen  *int64 `json:"last_seen,omitempty"`
	Timestamp int64  `json:"timestamp"`
}

// RoomJoined represents successful room join
type RoomJoined struct {
	Type         string   `json:"type"`
	RoomID       string   `json:"room_id"`
	UserID       string   `json:"user_id"`
	Participants []string `json:"participants,omitempty"`
	Timestamp    int64    `json:"timestamp"`
}

// RoomLeft represents room leave event
type RoomLeft struct {
	Type      string `json:"type"`
	RoomID    string `json:"room_id"`
	UserID    string `json:"user_id"`
	Timestamp int64  `json:"timestamp"`
}

// ErrorMessage represents error responses
type ErrorMessage struct {
	Type      string `json:"type"`
	Code      string `json:"code"`
	Message   string `json:"message"`
	Details   string `json:"details,omitempty"`
	Timestamp int64  `json:"timestamp"`
}

// SystemMessage represents system notifications
type SystemMessage struct {
	Type      string      `json:"type"`
	RoomID    string      `json:"room_id,omitempty"`
	Content   string      `json:"content"`
	Data      interface{} `json:"data,omitempty"`
	Timestamp int64       `json:"timestamp"`
}

type ReplyMessage struct {
	MessageID string `json:"messageId"`
	Content   string `json:"content"`
	SenderID  string `json:"senderId"`
}

type MessageEditEntry struct {
	MessageID       string `json:"message_id"`
	OriginalContent string `json:"original_content"`
	NewContent      string `json:"new_content"`
	EditedBy        string `json:"edited_by"`
	EditedAt        int64  `json:"edited_at"`
}

// MessageAttachment represents file attachments
type MessageAttachment struct {
	Type     string `json:"type"`
	URL      string `json:"url"`
	Filename string `json:"filename,omitempty"`
	Size     int64  `json:"size,omitempty"`
	MimeType string `json:"mime_type,omitempty"`
}

// Message type constants
const (
	// Outgoing message types (server -> client)
	MessageTypeChatMessage    = "chat_message"
	MessageTypeMessageUpdated = "message_updated"
	MessageTypeMessageRead    = "message_read"
	MessageTypeMessageDeleted = "message_deleted"
	MessageTypeUserTyping     = "user_typing"
	MessageTypeUserStatus     = "user_status"
	MessageTypeRoomJoined     = "room_joined"
	MessageTypeRoomLeft       = "room_left"
	MessageTypeError          = "error"
	MessageTypeSystem         = "system"
	MessageTypePong           = "pong"

	// Incoming message types (client -> server)
	MessageTypeJoinRoom    = "join_room"
	MessageTypeLeaveRoom   = "leave_room"
	MessageTypeTypingStart = "typing_start"
	MessageTypeTypingStop  = "typing_stop"
	MessageTypePing        = "ping"

	// User status constants
	UserStatusOnline  = "online"
	UserStatusOffline = "offline"
	UserStatusAway    = "away"

	// Error codes
	ErrorCodeInvalidMessage    = "INVALID_MESSAGE"
	ErrorCodeUnauthorized      = "UNAUTHORIZED"
	ErrorCodeRoomNotFound      = "ROOM_NOT_FOUND"
	ErrorCodeUserNotFound      = "USER_NOT_FOUND"
	ErrorCodeInternalError     = "INTERNAL_ERROR"
	ErrorCodeRateLimitExceeded = "RATE_LIMIT_EXCEEDED"
)

// NewChatMessage creates a new chat message
func NewChatMessage(roomID, messageID, senderID, content string) OutgoingMessage {
	return OutgoingMessage{
		Type:      MessageTypeChatMessage,
		RoomID:    roomID,
		MessageID: messageID,
		SenderID:  senderID,
		Data: ChatMessage{
			Type:      MessageTypeChatMessage,
			RoomID:    roomID,
			MessageID: messageID,
			SenderID:  senderID,
			Content:   content,
			CreatedAt: time.Now().Unix(),
			Timestamp: time.Now().Unix(),
		},
		Timestamp: time.Now().Unix(),
	}
}

// NewMessageUpdated creates a message updated notification
func NewMessageUpdated(roomID, messageID, content, editedBy string, editHistory []MessageEditEntry) OutgoingMessage {
	return OutgoingMessage{
		Type:      MessageTypeMessageUpdated,
		RoomID:    roomID,
		MessageID: messageID,
		SenderID:  editedBy,
		Data: MessageUpdated{
			Type:               MessageTypeMessageUpdated,
			RoomID:             roomID,
			MessageID:          messageID,
			Content:            content,
			IsEdited:           true,
			MessageEditHistory: editHistory,
			UpdatedAt:          time.Now().Unix(),
			EditedBy:           editedBy,
			Timestamp:          time.Now().Unix(),
		},
		Timestamp: time.Now().Unix(),
	}
}

// NewMessageRead creates a message read notification
func NewMessageRead(roomID, messageID, readBy string) OutgoingMessage {
	return OutgoingMessage{
		Type:      MessageTypeMessageRead,
		RoomID:    roomID,
		MessageID: messageID,
		Data: MessageRead{
			Type:      MessageTypeMessageRead,
			RoomID:    roomID,
			MessageID: messageID,
			ReadBy:    readBy,
			ReadAt:    time.Now().Unix(),
			Timestamp: time.Now().Unix(),
		},
		Timestamp: time.Now().Unix(),
	}
}

// NewUserTyping creates a typing indicator message
func NewUserTyping(roomID, userID string, isTyping bool) OutgoingMessage {
	return OutgoingMessage{
		Type:     MessageTypeUserTyping,
		RoomID:   roomID,
		SenderID: userID,
		Data: UserTyping{
			Type:      MessageTypeUserTyping,
			RoomID:    roomID,
			UserID:    userID,
			IsTyping:  isTyping,
			Timestamp: time.Now().Unix(),
		},
		Timestamp: time.Now().Unix(),
	}
}

// NewUserStatus creates a user status message
func NewUserStatus(roomID, userID, status string, lastSeen *time.Time) OutgoingMessage {
	var lastSeenUnix *int64
	if lastSeen != nil {
		ts := lastSeen.Unix()
		lastSeenUnix = &ts
	}

	return OutgoingMessage{
		Type:     MessageTypeUserStatus,
		RoomID:   roomID,
		SenderID: userID,
		Data: UserStatus{
			Type:      MessageTypeUserStatus,
			RoomID:    roomID,
			UserID:    userID,
			Status:    status,
			LastSeen:  lastSeenUnix,
			Timestamp: time.Now().Unix(),
		},
		Timestamp: time.Now().Unix(),
	}
}

// NewRoomJoined creates a room joined message
func NewRoomJoined(roomID, userID string, participants []string) OutgoingMessage {
	return OutgoingMessage{
		Type:     MessageTypeRoomJoined,
		RoomID:   roomID,
		SenderID: userID,
		Data: RoomJoined{
			Type:         MessageTypeRoomJoined,
			RoomID:       roomID,
			UserID:       userID,
			Participants: participants,
			Timestamp:    time.Now().Unix(),
		},
		Timestamp: time.Now().Unix(),
	}
}

// NewErrorMessage creates an error message
func NewErrorMessage(code, message, details string) OutgoingMessage {
	return OutgoingMessage{
		Type: MessageTypeError,
		Data: ErrorMessage{
			Type:      MessageTypeError,
			Code:      code,
			Message:   message,
			Details:   details,
			Timestamp: time.Now().Unix(),
		},
		Timestamp: time.Now().Unix(),
	}
}

// NewSystemMessage creates a system message
func NewSystemMessage(roomID, content string, data interface{}) OutgoingMessage {
	return OutgoingMessage{
		Type:   MessageTypeSystem,
		RoomID: roomID,
		Data: SystemMessage{
			Type:      MessageTypeSystem,
			RoomID:    roomID,
			Content:   content,
			Data:      data,
			Timestamp: time.Now().Unix(),
		},
		Timestamp: time.Now().Unix(),
	}
}

// Validation helpers

// IsValidMessageType checks if a message type is valid
func IsValidMessageType(msgType string) bool {
	validTypes := map[string]bool{
		MessageTypeChatMessage:    true,
		MessageTypeMessageUpdated: true,
		MessageTypeMessageRead:    true,
		MessageTypeMessageDeleted: true,
		MessageTypeUserTyping:     true,
		MessageTypeUserStatus:     true,
		MessageTypeRoomJoined:     true,
		MessageTypeRoomLeft:       true,
		MessageTypeError:          true,
		MessageTypeSystem:         true,
		MessageTypePong:           true,
		MessageTypeJoinRoom:       true,
		MessageTypeLeaveRoom:      true,
		MessageTypeTypingStart:    true,
		MessageTypeTypingStop:     true,
		MessageTypePing:           true,
	}
	return validTypes[msgType]
}

// IsValidUserStatus checks if a user status is valid
func IsValidUserStatus(status string) bool {
	return status == UserStatusOnline || status == UserStatusOffline || status == UserStatusAway
}
