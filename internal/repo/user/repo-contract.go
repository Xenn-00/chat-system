package user_repo

import (
	"context"

	"github.com/xenn00/chat-system/internal/entity"
	app_error "github.com/xenn00/chat-system/internal/errors"
)

type UserRepoContract interface {
	CountUser(ctx context.Context, filter entity.UserFilter) (int64, *app_error.AppError)
	SaveUser(ctx context.Context, model entity.User) *app_error.AppError
	VerifyUser(ctx context.Context, userId string) (*entity.User, *app_error.AppError)
}
