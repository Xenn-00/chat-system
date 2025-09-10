package entity

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Message struct {
	ID                 primitive.ObjectID  `bson:"_id,omitempty"`
	RoomID             string              `bson:"room_id"`
	SenderID           string              `bson:"sender_id"`
	ReceiverID         string              `bson:"receiver_id"`
	Content            string              `bson:"content"`
	IsRead             bool                `bson:"is_read"`
	IsEdited           bool                `bson:"is_edited"`
	MessageEditHistory []*MessageEditEntry `bson:"message_edit_history"`
	Attachments        []*Attachment       `bson:"attachments"`
	ReplyTo            *ReplyTo            `bson:"reply_to"`
	CreatedAt          time.Time           `bson:"created_at"`
	UpdatedAt          *time.Time          `bson:"updated_at"`
}

type MessageEditEntry struct {
	MessageID       primitive.ObjectID `bson:"message_id"`
	OriginalContent string             `bson:"original_content"`
	NewContent      string             `bson:"new_content"`
	EditedBy        string             `bson:"edited_by"`
	EditedAt        time.Time          `bson:"edited_at"`
}

type ReplyTo struct {
	MessageID primitive.ObjectID `bson:"message_id"`
	Content   string             `bson:"content"`
	SenderID  string             `bson:"sender_id"`
}

type Attachment struct {
	Type string `bson:"type"`
	URL  string `bson:"url"`
}
