package chat_dto

type SendPrivateMessageRequest struct {
	Content string `json:"content" validate:"required,min=1"`
}

type GetPrivateMessagesRequest struct {
	Limit    int     `json:"limit" validate:"omitempty,min=1,max=100"`
	BeforeID *string `json:"before_id,omitempty" query:"before_id"` // for cursor pagination
}

type ReplyPrivateMessageRequest struct {
	Content    string `json:"content" validate:"required,min=1"`
	ReplyTo    string `json:"reply_to" validate:"required"` // message ID being replied to
	ReceiverID string `json:"receiver_id" validate:"required,uuid"`
}
