package chat_dto

import "time"

type CreateRoomResponse struct {
	RoomID    string    `json:"room_id"`
	RoomType  string    `json:"room_type"`
	CreatedBy string    `json:"created_by"` // return username of the creator
	CreatedAt time.Time `json:"created_at"`
}

type SendPrivateMessageResponse struct {
	MessageID string    `json:"message_id"`
	RoomID    string    `json:"room_id"`
	SenderID  string    `json:"sender_id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}
