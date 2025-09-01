package chat_dto

import "time"

type SendPrivateMessageResponse struct {
	MessageID string    `json:"message_id"`
	RoomID    string    `json:"room_id"`
	SenderID  string    `json:"sender_id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

type ReplyPrivateMessageResponse struct {
	MessageID  string       `json:"message_id"`
	RoomID     string       `json:"room_id"`
	SenderID   string       `json:"sender_id"`
	ReceiverID string       `json:"receiver_id"`
	Content    string       `json:"content"`
	ReplyTo    ReplyMessage `json:"reply_to"`
	CreatedAt  time.Time    `json:"created_at"`
}

type ReplyMessage struct {
	MessageID string `json:"message_id"`
	Content   string `json:"content"`
	SenderID  string `json:"sender_id"`
}

type GetPrivateMessagesResponse struct {
	Messages   []PrivateMessages `json:"messages"`
	NextCursor *string           `json:"next_cursor,omitempty"`
	HasMore    bool              `json:"has_more"`
}

type PrivateMessages struct {
	MessageID  string    `json:"message_id"`
	SenderID   string    `json:"sender_id"`
	ReceiverID string    `json:"receiver_id"`
	Content    string    `json:"content"`
	IsRead     bool      `json:"is_read"`
	CreatedAt  time.Time `json:"created_at"`
}
