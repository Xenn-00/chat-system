package user_handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/xenn00/chat-system/internal/dtos/user_dto"
	app_error "github.com/xenn00/chat-system/internal/errors"
	"github.com/xenn00/chat-system/internal/handlers"
	"github.com/xenn00/chat-system/internal/middleware"
	"github.com/xenn00/chat-system/internal/queue"
	user_service "github.com/xenn00/chat-system/internal/use-case/user-case"
	"github.com/xenn00/chat-system/state"
)

type UserHandler struct {
	State    *state.AppState
	Producer queue.Producer
	Validate *validator.Validate
	Service  user_service.UserServiceContract
}

func NewUserHandler(state *state.AppState) *UserHandler {
	validate := validator.New()
	_ = validate.RegisterValidation("otpval", user_dto.OTPValidator)
	return &UserHandler{
		State:    state,
		Producer: queue.NewProducer(state.Redis),
		Validate: validate,
		Service:  user_service.NewUserService(state),
	}
}

func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) *app_error.AppError {
	var req user_dto.CreateUserRequest
	defer r.Body.Close()

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return app_error.NewAppError(http.StatusBadRequest, "Invalid JSON", "body")
	}

	if err := h.Validate.Struct(req); err != nil {
		return app_error.NewAppError(http.StatusBadRequest, fmt.Sprintf("Invalid fields: %v", err), "validation")
	}

	resp, err := h.Service.Register(r.Context(), req)
	if err != nil {
		return err
	}

	reqID, ok := r.Context().Value(middleware.RequestIdKey).(string)
	if !ok {
		reqID = "unknown"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(handlers.CreateResponse("user registered successfully", *resp, reqID))

	go func() {
		reqJob := map[string]any{
			"user_id":    resp.ID,
			"created_at": resp.CreatedAt,
		}

		job := queue.Job{
			ID:        uuid.New().String(),
			Type:      "create_user_otp",
			Payload:   queue.MustMarshal(reqJob),
			Priority:  1,
			Retry:     0,
			MaxRetry:  5,
			CreatedAt: time.Now().Unix(),
			ExpireAt:  time.Now().Add(5 * time.Minute).Unix(),
		}

		if err := h.Producer.Enqueue(h.State.Ctx, job); err != nil {
			log.Error().Err(err).Msg("Failed to enqueue job")
		}
	}()

	return nil
}

func (h *UserHandler) VerifyUser(w http.ResponseWriter, r *http.Request) *app_error.AppError {
	var req user_dto.VerifyUserRequest
	defer r.Body.Close()

	user_id := chi.URLParam(r, "userId")

	log.Info().Msgf("user_id: %s", user_id)

	// get fingerprint
	fp := r.Context().Value(middleware.FingerprintKey).(string)
	if fp == "" {
		return app_error.NewAppError(http.StatusBadRequest, "Missing device fingerprint", "fingerprint")
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return app_error.NewAppError(http.StatusBadRequest, "Invalid JSON", "body")
	}

	if err := h.Validate.Struct(req); err != nil {
		return app_error.NewAppError(http.StatusBadRequest, fmt.Sprintf("Invalid fields: %v", err), "validation")
	}

	resp, err := h.Service.VerifyRegister(r.Context(), req, fp, user_id)
	if err != nil {
		return err
	}

	if len(resp.Refresh) == 0 {
		return app_error.NewAppError(http.StatusInternalServerError, "failed to prepare refresh token", "server")
	}

	reqID, ok := r.Context().Value(middleware.RequestIdKey).(string)
	if !ok {
		reqID = "unknown"
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    resp.Refresh,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
		Expires:  time.Now().Add(7 * 24 * time.Hour),
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(handlers.CreateResponse("user registered successfully", *resp, reqID))

	return nil
}

func (h *UserHandler) LoginUser(w http.ResponseWriter, r *http.Request) *app_error.AppError {
	var req user_dto.LoginUserRequest
	defer r.Body.Close()

	// get fingerprint
	fp := r.Context().Value(middleware.FingerprintKey).(string)
	if fp == "" {
		return app_error.NewAppError(http.StatusBadRequest, "Missing device fingerprint", "fingerprint")
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return app_error.NewAppError(http.StatusBadRequest, "Invalid JSON", "body")
	}

	if err := h.Validate.Struct(req); err != nil {
		return app_error.NewAppError(http.StatusBadRequest, fmt.Sprintf("Invalid fields: %v", err), "validation")
	}

	resp, err := h.Service.Login(r.Context(), req, fp)
	if err != nil {
		return err
	}

	if len(resp.Refresh) == 0 {
		return app_error.NewAppError(http.StatusInternalServerError, "failed to prepare refresh token", "server")
	}

	reqID, ok := r.Context().Value(middleware.RequestIdKey).(string)
	if !ok {
		reqID = "unknown"
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    resp.Refresh,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
		Expires:  time.Now().Add(7 * 24 * time.Hour),
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(handlers.CreateResponse("user registered successfully", *resp, reqID))

	return nil
}
