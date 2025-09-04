package websocket

type Message struct {
	Type      string        `json:"type"`
	RoomId    string        `json:"roomId"`
	SenderID  string        `json:"senderId"`
	Content   string        `json:"content"`
	Reply     *ReplyMessage `json:"reply,omitempty"`
	Timestamp int64         `json:"timestamp"`
}

type ReplyMessage struct {
	MessageID string `json:"messageId"`
	Content   string `json:"content"`
	SenderID  string `json:"senderId"`
}
