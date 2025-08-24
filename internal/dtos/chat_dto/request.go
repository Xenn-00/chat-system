package chat_dto

type CreateRoomRequest struct {
	RoomType string  `json:"room_type" validate:"required"`
	Name     *string `json:"name" validate:"omitempty"`
}

type SendPrivateMessageRequest struct {
	ReceiverID string `json:"receiverId" validate:"required,uuid4"`
	Content    string `json:"content" validate:"required,min=1"`
}
