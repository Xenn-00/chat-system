package chat_dto

type WSIncomingMessage struct {
	Type    string `json:"type"`
	RoomID  string `json:"roomId"`
	Content string `json:"content,omitempty"`
}
