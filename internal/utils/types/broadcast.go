package types

import (
	"time"
)

type BroadcastMessagePayload struct {
	MessageID          string              `json:"message_id"`
	RoomID             string              `json:"room_id"`
	SenderID           string              `json:"sender_id"`
	ReceiverID         string              `json:"receiver_id"`
	Content            string              `json:"content"`
	IsRead             *bool               `json:"is_read"`
	IsEdited           *bool               `json:"is_edited"`
	MessageEditHistory []*MessageEditEntry `json:"message_edit_history"`
	Attachments        []*Attachment       `json:"attachments"`
	ReplyTo            *ReplyTo            `json:"reply_to"`
	CreatedAt          time.Time           `json:"created_at"`
	UpdatedAt          *time.Time          `json:"updated_at"`
}

type MessageEditEntry struct {
	MessageID       string    `json:"message_id"`
	OriginalContent string    `json:"original_content"`
	NewContent      string    `json:"new_content"`
	EditedBy        string    `json:"edited_by"`
	EditedAt        time.Time `json:"edited_at"`
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
