package chat_dto

type WSOutgoingMessage struct {
	Event     string `json:"event"`
	RoomID    string `json:"roomId"`
	SenderID  string `json:"senderId"`
	Content   string `json:"content"`
	Timestamp int64  `json:"timestamp"`
}
