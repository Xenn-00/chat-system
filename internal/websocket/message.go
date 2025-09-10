package websocket

type Message struct {
	Type               string              `json:"type"`
	RoomId             string              `json:"roomId"`
	SenderID           string              `json:"senderId"`
	Content            string              `json:"content"`
	MessageEditHistory []*MessageEditEntry `json:"message_edit_history"`
	Reply              *ReplyMessage       `json:"reply,omitempty"`
	Timestamp          int64               `json:"timestamp"`
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
