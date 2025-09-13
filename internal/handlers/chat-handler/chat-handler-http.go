package chat_handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog/log"
	"github.com/xenn00/chat-system/internal/dtos/chat_dto"
	app_error "github.com/xenn00/chat-system/internal/errors"
	"github.com/xenn00/chat-system/internal/handlers"
	"github.com/xenn00/chat-system/internal/middleware"
	"github.com/xenn00/chat-system/internal/queue"
	chat_service "github.com/xenn00/chat-system/internal/use-case/chat-case"
	"github.com/xenn00/chat-system/state"
)

type ChatHandler struct {
	State    *state.AppState
	Producer queue.Producer
	Validate *validator.Validate
	Service  chat_service.ChatServiceContract
}

func NewChatHandler(state *state.AppState) *ChatHandler {
	validate := validator.New()
	validate.RegisterValidation("objectID", chat_dto.ObjectIDValidator)
	return &ChatHandler{
		State:    state,
		Producer: queue.NewProducer(state.Redis),
		Validate: validate,
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
	json.NewEncoder(w).Encode(handlers.CreateResponse("message sent successfully", *resp, reqID))

	// notif / ws broadcast
	go func() {
		if err := h.broadcastPrivateMessage(resp); err != nil {
			log.Error().Err(err).Msg("failed to broadcast message")
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
	json.NewEncoder(w).Encode(handlers.CreateResponse("messages fetch successfully", *resp, reqID))

	return nil

}

func (h *ChatHandler) ReplyPrivateMessage(w http.ResponseWriter, r *http.Request) *app_error.AppError {
	var req chat_dto.ReplyPrivateMessageRequest
	defer r.Body.Close()

	// I need to get room_id from uri param
	roomID := chi.URLParam(r, "roomId")

	// get user_id from context
	userID, ok := r.Context().Value(middleware.UserClaimsKey).(string)
	if !ok || userID == "" {
		return app_error.NewAppError(http.StatusUnauthorized, "user id is not found in context", "context")
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return app_error.NewAppError(http.StatusBadRequest, "Invalid JSON", "body")
	}

	if err := h.Validate.Struct(req); err != nil {
		return app_error.NewAppError(http.StatusBadRequest, fmt.Sprintf("Invalid fields: %v", err), "validation")
	}

	resp, err := h.Service.ReplyPrivateMessage(r.Context(), req, userID, roomID)
	if err != nil {
		return err
	}

	reqID, ok := r.Context().Value(middleware.RequestIdKey).(string)
	if !ok {
		reqID = "unknown"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(handlers.CreateResponse("message replied successfully", *resp, reqID))

	// notif / ws broadcast
	go func() {
		if err := h.broadcastPrivateMessageReply(resp); err != nil {
			log.Error().Err(err).Msg("failed to broadcast message reply")
		}
	}()

	return nil
}

func (h *ChatHandler) MarkMessageAsRead(w http.ResponseWriter, r *http.Request) *app_error.AppError {
	// get room_id from uri param and message_id from query param
	roomID := chi.URLParam(r, "roomId")
	messageID := r.URL.Query().Get("messageID")

	if roomID == "" || messageID == "" {
		return app_error.NewAppError(http.StatusBadRequest, "room_id and message_id are required", "params")
	}

	// get user_id from context
	userID, ok := r.Context().Value(middleware.UserClaimsKey).(string)
	if !ok || userID == "" {
		return app_error.NewAppError(http.StatusUnauthorized, "user id is not found in context", "context")
	}

	if err := h.Service.MarkPrivateMessageAsRead(r.Context(), userID, roomID, messageID); err != nil {
		return err
	}

	reqID, ok := r.Context().Value(middleware.RequestIdKey).(string)
	if !ok {
		reqID = "unknown"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(handlers.CreateResponse("message marked as read successfully", "OK", reqID))

	return nil
}

func (h *ChatHandler) UpdatePrivateMessage(w http.ResponseWriter, r *http.Request) *app_error.AppError {
	var req chat_dto.UpdatePrivateMessageRequest
	defer r.Body.Close()

	// get room_id from uri param and message_id from query param
	roomID := chi.URLParam(r, "roomId")
	messageID := r.URL.Query().Get("messageID")

	if roomID == "" || messageID == "" {
		return app_error.NewAppError(http.StatusBadRequest, "room_id and message_id are required", "params")
	}

	// get user id as sender id from context
	userID, ok := r.Context().Value(middleware.UserClaimsKey).(string)
	if !ok || userID == "" {
		return app_error.NewAppError(http.StatusUnauthorized, "user id is not found in context", "context")
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return app_error.NewAppError(http.StatusBadRequest, "Invalid JSON", "body")
	}

	if err := h.Validate.Struct(req); err != nil {
		return app_error.NewAppError(http.StatusBadRequest, fmt.Sprintf("Invalid fields: %v", err), "validation")
	}

	resp, err := h.Service.UpdatePrivateMessage(r.Context(), req, userID, roomID, messageID)
	if err != nil {
		return err
	}

	reqID, ok := r.Context().Value(middleware.RequestIdKey).(string)
	if !ok {
		reqID = "unknown"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(handlers.CreateResponse("message edited", *resp, reqID))

	// notif / ws broadcast
	go h.broadcastPrivateMessageUpdated(resp)

	return nil
}
