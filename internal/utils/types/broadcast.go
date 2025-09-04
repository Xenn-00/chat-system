package types

import (
	"time"
)

type BroadcastMessagePayload struct {
	MessageID   string        `json:"message_id"`
	RoomID      string        `json:"room_id"`
	SenderID    string        `json:"sender_id"`
	ReceiverID  string        `json:"receiver_id"`
	Content     string        `json:"content"`
	IsRead      *bool         `json:"is_read"`
	IsEdited    *bool         `json:"is_edited"`
	Attachments []*Attachment `json:"attachments"`
	ReplyTo     *ReplyTo      `json:"reply_to"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   *time.Time    `json:"updated_at"`
}

type ReplyTo struct {
	MessageID string `json:"message_id"`
	Content   string `json:"content"`
	SenderID  string `json:"sender_id"`
}

type Attachment struct {
	Type string `json:"type"`
	URL  string `json:"url"`
}
