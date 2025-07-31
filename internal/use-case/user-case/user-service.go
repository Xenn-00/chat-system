package user_service

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/xenn00/chat-system/internal/dtos/user_dto"
	"github.com/xenn00/chat-system/internal/entity"
	app_error "github.com/xenn00/chat-system/internal/errors"
	user_repo "github.com/xenn00/chat-system/internal/repo/user"
	"github.com/xenn00/chat-system/internal/utils"
	"github.com/xenn00/chat-system/state"
)

type UserService struct {
	AppState *state.AppState
	UserRepo user_repo.UserRepoContract
}

func NewUserService(appState *state.AppState) UserServiceContract {
	return &UserService{
		AppState: appState,
		UserRepo: user_repo.NewUserRepo(appState),
	}
}

func (u *UserService) Register(ctx context.Context, req user_dto.CreateUserRequest) (*user_dto.UserResponse, *app_error.AppError) {
	// count user, is the user already registered or not
	filter := &entity.UserFilter{
		Email:    &req.Email,
		Username: &req.Username,
	}
	count, err := u.UserRepo.CountUser(ctx, *filter)
	if err != nil {
		return nil, err
	}

	if count > 0 {
		return nil, app_error.NewAppError(http.StatusConflict, "username or email already registered", "credential-registered")
	}

	hashed, hashErr := utils.GenerateHash(req.Password)
	if hashErr != nil {
		return nil, app_error.NewAppError(http.StatusInternalServerError, hashErr.Error(), "password")
	}

	user := &entity.User{
		ID:           uuid.New().String(),
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: hashed,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	err = u.UserRepo.SaveUser(ctx, *user)
	if err != nil {
		return nil, err
	}

	return &user_dto.UserResponse{
		ID:        user.ID,
		Email:     user.Email,
		Username:  user.Username,
		CreatedAt: user.CreatedAt,
	}, nil
}

func (u *UserService) VerifyRegister(ctx context.Context, req user_dto.VerifyUserRequest, userId string) (bool, *app_error.AppError) {
	key := fmt.Sprintf("otp:%s", userId)
	otp, err := u.AppState.Redis.Get(ctx, key).Result()
	if err == redis.Nil {
		return false, app_error.NewAppError(http.StatusNotFound, "otp is expired or not found", "redis-otp")
	} else if err != nil {
		return false, app_error.NewAppError(http.StatusInternalServerError, "unexpected error occured when redis get otp", "redis-otp")
	}

	if otp != req.OTP {
		return false, app_error.NewAppError(http.StatusBadRequest, "otp mismatch", "otp-mismatch")
	}

	u.AppState.Redis.Del(ctx, key)

	verified, r_err := u.UserRepo.VerifyUser(ctx, userId)
	if r_err != nil {
		return false, r_err
	}

	return verified, nil
}
