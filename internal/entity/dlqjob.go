package entity

import (
	"encoding/json"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type DLQJob struct {
	ID                 primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	JobID              string             `bson:"job_id" json:"job_id"`
	Type               string             `bson:"type" json:"type"`
	Payload            json.RawMessage    `bson:"payload" json:"payload"`
	ErrorMsg           string             `bson:"error_msg" json:"error_msg"`
	Status             string             `bson:"status" json:"status"`
	RetryCount         int                `bson:"retry_count" json:"retry_count"`
	OriginalRetryCount int                `bson:"original_retry_count" json:"original_retry_count"`
	NextRetryAt        *time.Time         `bson:"next_retry_at,omitempty" json:"next_retry_at,omitempty"`
	CreatedAt          time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt          time.Time          `bson:"updated_at" json:"updated_at"`
	CompletedAt        *time.Time         `bson:"completed_at,omitempty" json:"completed_at,omitempty"`
	FailedAt           *time.Time         `bson:"failed_at,omitempty" json:"failed_at,omitempty"`
	ExpireAt           time.Time          `bson:"expired_at" json:"expired_at"`
}
