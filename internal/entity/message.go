package entity

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Message struct {
	ID       primitive.ObjectID `bson:"_id,omitempty"`
	RoomID   string             `bson:"roomId"`
	SenderID string             `bson:"senderId"`
	Content  string             `bson:"content"`
	IsRead   bool               `bson:"isRead"`
	CreateAt time.Time          `bson:"createdAt"`
}
