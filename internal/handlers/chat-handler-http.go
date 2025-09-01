package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/xenn00/chat-system/internal/dtos/chat_dto"
	app_error "github.com/xenn00/chat-system/internal/errors"
	"github.com/xenn00/chat-system/internal/middleware"
	"github.com/xenn00/chat-system/internal/queue"
	chat_service "github.com/xenn00/chat-system/internal/use-case/chat-case"
	"github.com/xenn00/chat-system/internal/utils/types"
	"github.com/xenn00/chat-system/state"
)

type ChatHandler struct {
	State    *state.AppState
	Producer queue.Producer
	Validate *validator.Validate
	Service  chat_service.ChatServiceContract
}

func NewChatHandler(state *state.AppState) *ChatHandler {
	return &ChatHandler{
		State:    state,
		Producer: queue.NewProducer(state.Redis),
		Validate: validator.New(),
		Service:  chat_service.NewChatService(state),
	}
}

func (h *ChatHandler) SendPrivateMessage(w http.ResponseWriter, r *http.Request) *app_error.AppError {
	var req chat_dto.SendPrivateMessageRequest
	defer r.Body.Close()

	// get receiver_id from uri param
	receiverID := chi.URLParam(r, "receiverId")
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return app_error.NewAppError(http.StatusBadRequest, "Invalid JSON", "body")
	}

	if err := h.Validate.Struct(req); err != nil {
		return app_error.NewAppError(http.StatusBadRequest, fmt.Sprintf("Invalid fields: %v", err), "validation")
	}

	userID, ok := r.Context().Value(middleware.UserClaimsKey).(string)
	if !ok || userID == "" {
		return app_error.NewAppError(http.StatusUnauthorized, "user id is not found in context", "context")
	}

	resp, err := h.Service.SendPrivateMessage(r.Context(), req, userID, receiverID)
	if err != nil {
		return err
	}

	reqID, ok := r.Context().Value(middleware.RequestIdKey).(string)
	if !ok {
		reqID = "unknown"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(CreateResponse("message sent successfully", *resp, reqID))

	// notif / ws broadcast
	go func() {
		jobPayload := &types.BroadcastPayload{
			RoomID:     resp.RoomID,
			MessageID:  resp.MessageID,
			SenderID:   resp.SenderID,
			ReceiverID: receiverID,
			Content:    resp.Content,
			CreatedAt:  resp.CreatedAt,
		}

		job := queue.Job{
			ID:        uuid.New().String(),
			Type:      "broadcast_private_message",
			Payload:   queue.MustMarshal(jobPayload),
			Priority:  1,
			Retry:     0,
			MaxRetry:  3,
			CreatedAt: time.Now().Unix(),
			ExpireAt:  time.Now().Add(1 * time.Minute).Unix(),
		}

		if err := h.Producer.Enqueue(h.State.Ctx, job); err != nil {
			log.Error().Err(err).Msg("Failed to enqueue job")
		}
	}()

	return nil
}

func (h *ChatHandler) GetPrivateMessages(w http.ResponseWriter, r *http.Request) *app_error.AppError {
	var req chat_dto.GetPrivateMessagesRequest
	defer r.Body.Close()

	// get room_id from query param
	roomID := chi.URLParam(r, "roomId")

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return app_error.NewAppError(http.StatusBadRequest, fmt.Sprintf("Invalid JSON: %v", err), "body")
	}

	if err := h.Validate.Struct(req); err != nil {
		return app_error.NewAppError(http.StatusBadRequest, fmt.Sprintf("Invalid fields: %v", err), "validation")
	}

	resp, err := h.Service.GetPrivateMessage(r.Context(), req, roomID)
	if err != nil {
		return err
	}

	reqID, ok := r.Context().Value(middleware.RequestIdKey).(string)
	if !ok {
		reqID = "unknown"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(CreateResponse("messages fetch successfully", *resp, reqID))

	return nil
}
