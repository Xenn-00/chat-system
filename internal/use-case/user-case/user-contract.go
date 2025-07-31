package user_service

import (
	"context"

	"github.com/xenn00/chat-system/internal/dtos/user_dto"
	app_error "github.com/xenn00/chat-system/internal/errors"
)

type UserServiceContract interface {
	Register(ctx context.Context, req user_dto.CreateUserRequest) (*user_dto.UserResponse, *app_error.AppError)
	VerifyRegister(ctx context.Context, req user_dto.VerifyUserRequest, userId string) (bool, *app_error.AppError)
}
