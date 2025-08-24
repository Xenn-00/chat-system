package user_dto

import "time"

type UserResponse struct {
	ID         string    `json:"id"`
	Username   string    `json:"username"`
	Email      string    `json:"email"`
	IsVerified bool      `json:"is_verified"`
	Token      *string   `json:"token"`
	Refresh    *string   `json:"refresh"`
	CreatedAt  time.Time `json:"created_at"`
}
