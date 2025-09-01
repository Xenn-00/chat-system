package user_repo

import (
	"context"
	"errors"
	"net/http"

	"github.com/xenn00/chat-system/internal/entity"
	app_error "github.com/xenn00/chat-system/internal/errors"
	"github.com/xenn00/chat-system/state"
	"gorm.io/gorm"
)

type UserRepo struct {
	AppState *state.AppState
}

func NewUserRepo(appState *state.AppState) UserRepoContract {
	return &UserRepo{
		AppState: appState,
	}
}

func (r *UserRepo) CountUser(ctx context.Context, filter entity.UserFilter) (int64, *app_error.AppError) {
	var count int64

	query := r.AppState.DB.WithContext(ctx).Model(&entity.User{})

	if filter.Email != nil {
		query = query.Where("email = ?", filter.Email)
	}

	if filter.Username != nil {
		query = query.Where("username = ?", filter.Username)
	}

	if filter.Email != nil && filter.Username != nil {
		query = query.Where("email = ? AND username = ?", filter.Email, filter.Username)
	}

	err := query.Count(&count).Error
	if err != nil {
		if errors.Is(err, gorm.ErrEmptySlice) {
			return 0, nil
		}
		return 0, app_error.NewAppError(http.StatusInternalServerError, "unexpected server error", "db-count")
	}
	return count, nil
}

func (r *UserRepo) SaveUser(ctx context.Context, model entity.User) *app_error.AppError {
	if err := r.AppState.DB.WithContext(ctx).Create(model).Error; err != nil {
		return app_error.NewAppError(http.StatusInternalServerError, "unexpected error occur when trying to create user", "db-create")
	}

	return nil
}

func (r *UserRepo) VerifyUser(ctx context.Context, userId string) (*entity.User, *app_error.AppError) {
	var user entity.User

	if err := r.AppState.DB.WithContext(ctx).Where("id = ?", userId).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, app_error.NewAppError(http.StatusNotFound, "cannot find user", "user-id")
		}
		return nil, app_error.NewAppError(http.StatusInternalServerError, "unexpected error occur when fetch user", "db-error")
	}

	user.IsActive = true

	if err := r.AppState.DB.WithContext(ctx).Where("id = ?", userId).Updates(user).Error; err != nil {
		return nil, app_error.NewAppError(http.StatusInternalServerError, "unexpected error occured when verifying user", "db-update")
	}

	return &user, nil
}

func (r *UserRepo) FindUserByCredential(ctx context.Context, username string) (*entity.User, *app_error.AppError) {
	var user entity.User

	if err := r.AppState.DB.WithContext(ctx).Where("username = ?", username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, app_error.NewAppError(http.StatusNotFound, "cannot find user", "user-credential")
		}
		return nil, app_error.NewAppError(http.StatusInternalServerError, "unexpected error occur when fetch user", "db-error")
	}

	return &user, nil
}
