package user_service

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/xenn00/chat-system/internal/dtos/user_dto"
	"github.com/xenn00/chat-system/internal/entity"
	app_error "github.com/xenn00/chat-system/internal/errors"
	user_repo "github.com/xenn00/chat-system/internal/repo/user"
	"github.com/xenn00/chat-system/internal/utils"
	"github.com/xenn00/chat-system/internal/utils/types"
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

func (u *UserService) VerifyRegister(ctx context.Context, req user_dto.VerifyUserRequest, fingerprint string, userId string) (*user_dto.UserResponse, *app_error.AppError) {
	key := fmt.Sprintf("otp:%s", userId)
	// otp, err := u.AppState.Redis.Get(ctx, key).Result()
	otp, err := utils.GetCacheData[string](ctx, u.AppState.Redis, key)
	if err != nil {
		return nil, err
	}

	if *otp != req.OTP {
		return nil, app_error.NewAppError(http.StatusBadRequest, "otp mismatch", "otp-mismatch")
	}

	if err := u.AppState.Redis.Del(ctx, key).Err(); err != nil {
		log.Error().Err(err).Msg("failed to delete otp key")
	}

	user, r_err := u.UserRepo.VerifyUser(ctx, userId)
	if r_err != nil {
		return nil, r_err
	}

	issue_at := time.Now().Unix()
	expires_refresh := issue_at + 7*24*3600 // a week

	access, refresh, jti, e := utils.IssueNewTokens(userId, user.Username, u.AppState.JwtSecret.Private)
	if e != nil {
		log.Error().Err(e).Msg("error occured when signing token")
		return nil, app_error.NewAppError(http.StatusInternalServerError, fmt.Sprintf("unexpected error occured when sign token: %v", e), "token-sign")
	}

	refreshSessionKey := fmt.Sprintf("refresh:%s:%s:%s", userId, fingerprint, jti)
	session := &types.RefreshSession{
		UserId:      userId,
		JTI:         jti,
		Fingerprint: fingerprint,
		IssueAt:     issue_at,
		ExpireAt:    expires_refresh,
		Status:      "valid",
	}

	utils.SetCacheData(ctx, u.AppState.Redis, refreshSessionKey, session, time.Duration(time.Until(time.Unix(expires_refresh, 0))))

	userSessionsKey := fmt.Sprintf("sessions:%s", userId)
	u.AppState.Redis.SAdd(ctx, userSessionsKey, jti)
	u.AppState.Redis.ExpireAt(ctx, userSessionsKey, time.Unix(expires_refresh, 0))

	return &user_dto.UserResponse{
		ID:         userId,
		IsVerified: user.IsActive,
		Token:      &access,
		Refresh:    &refresh,
	}, nil
}
