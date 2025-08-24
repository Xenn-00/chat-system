package entity

import (
	"time"

	"github.com/google/uuid"
)

type Room struct {
	ID        uuid.UUID `gorm:"primaryKey"`
	RT        string    `gorm:"not null"`
	Name      string    `gorm:"not null"`
	CreatedBy string    `gorm:"not null"`
	CreatedAt time.Time `gorm:"not null"`
	UpdatedAt time.Time `gorm:"autoCreateTime"`
	DeletedAt time.Time `gorm:"autoUpdateTime"`
}

type RoomMember struct {
	ID            int64     `gorm:"primaryKey"`
	RoomID        string    `gorm:"not null"`
	UserID        string    `gorm:"not null"`
	Role          string    `gorm:"not null"`
	JoinedAt      time.Time `gorm:"autoCreateTime"`
	LeftAt        time.Time
	LastReadMsgID string
	LastMessageAt time.Time
	UnreadCount   int64
}
