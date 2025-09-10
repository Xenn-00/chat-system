package chat_dto

import (
	"github.com/go-playground/validator/v10"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type SendPrivateMessageRequest struct {
	Content string `json:"content" validate:"required,min=1"`
}

type GetPrivateMessagesRequest struct {
	Limit    int     `json:"limit" validate:"omitempty,min=1,max=100"`
	BeforeID *string `json:"before_id,omitempty" query:"before_id"` // for cursor pagination
}

type ReplyPrivateMessageRequest struct {
	Content    string `json:"content" validate:"required,min=1"`
	ReplyTo    string `json:"reply_to" validate:"required,objectID"` // message ID being replied to
	ReceiverID string `json:"receiver_id" validate:"required,uuid"`
}

type UpdatePrivateMessageRequest struct {
	Content string `json:"content" validate:"min=1"`
}

func ObjectIDValidator(fl validator.FieldLevel) bool {
	id := fl.Field().String()
	_, err := primitive.ObjectIDFromHex(id)
	return err == nil
}
