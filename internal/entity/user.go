package entity

import (
	"time"
)

type User struct {
	ID           string    `gorm:"primaryKey"`
	Username     string    `gorm:"uniqueIndex"`
	Email        string    `gorm:"uniqueIndex"`
	PasswordHash string    `gorm:"not null"`
	IsActive     bool      `gorm:"not null"`
	CreatedAt    time.Time `gorm:"autoCreateTime"`
	UpdatedAt    time.Time `gorm:"autoUpdateTime"`
}

type UserFilter struct {
	Email    *string
	Username *string
}
