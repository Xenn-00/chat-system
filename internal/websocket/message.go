package websocket

type Message struct {
	Type      string `json:"type"`
	RoomId    string `json:"roomId"`
	SenderID  string `json:"senderId"`
	Content   string `json:"content"`
	Timestamp int64  `json:"timestamp"`
}
