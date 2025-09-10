package chat_dto

import "time"

type SendPrivateMessageResponse struct {
	MessageID  string    `json:"message_id"`
	RoomID     string    `json:"room_id"`
	SenderID   string    `json:"sender_id"`
	ReceiverID string    `json:"receiver_id"`
	Content    string    `json:"content"`
	IsRead     bool      `json:"is_read"`
	CreatedAt  time.Time `json:"created_at"`
}

type UpdatePrivateMessageResponse struct {
	MessageID          string              `json:"message_id"`
	RoomID             string              `json:"room_id"`
	SenderID           string              `json:"sender_id"`
	ReceiverID         string              `json:"receiver_id"`
	Content            string              `json:"content"`
	MessageEditHistory []*MessageEditEntry `json:"message_edit_history"`
	ReplyTo            *ReplyMessage       `json:"reply_to"`
	IsRead             bool                `json:"is_read"`
	IsEdited           bool                `json:"is_edited"`
	UpdatedAt          time.Time           `json:"updated_at"`
}

type MessageEditEntry struct {
	MessageID       string    `json:"message_id"`
	OriginalContent string    `json:"original_content"`
	NewContent      string    `json:"new_content"`
	EditedBy        string    `json:"edited_by"`
	EditedAt        time.Time `json:"edited_at"`
}

type ReplyPrivateMessageResponse struct {
	MessageID  string        `json:"message_id"`
	RoomID     string        `json:"room_id"`
	SenderID   string        `json:"sender_id"`
	ReceiverID string        `json:"receiver_id"`
	Content    string        `json:"content"`
	ReplyTo    *ReplyMessage `json:"reply_to"`
	IsRead     bool          `json:"is_read"`
	CreatedAt  time.Time     `json:"created_at"`
}

type ReplyMessage struct {
	RepliedMessageID string `json:"message_id"`
	Content          string `json:"content"`
	SenderID         string `json:"sender_id"`
}

type GetPrivateMessagesResponse struct {
	Messages   []PrivateMessages `json:"messages"`
	NextCursor *string           `json:"next_cursor,omitempty"`
	HasMore    bool              `json:"has_more"`
}

type PrivateMessages struct {
	MessageID  string        `json:"message_id"`
	RoomID     string        `json:"room_id"`
	SenderID   string        `json:"sender_id"`
	ReceiverID string        `json:"receiver_id"`
	Content    string        `json:"content"`
	ReplyTo    *ReplyMessage `json:"reply_to,omitempty"`
	IsRead     bool          `json:"is_read"`
	CreatedAt  time.Time     `json:"created_at"`
}
