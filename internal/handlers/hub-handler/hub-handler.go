package hub_handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	app_error "github.com/xenn00/chat-system/internal/errors"
	"github.com/xenn00/chat-system/internal/handlers"
	"github.com/xenn00/chat-system/internal/middleware"
	"github.com/xenn00/chat-system/internal/websocket"
)

type HubHandler struct {
	Hub *websocket.Hub
}

func NewHubHandler(hub *websocket.Hub) *HubHandler {
	return &HubHandler{
		Hub: hub,
	}
}

func (h *HubHandler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]any{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
		"service":   "websocket-server",
	})
}

func (h *HubHandler) HandleGetStats(w http.ResponseWriter, r *http.Request) *app_error.AppError {
	stats := h.Hub.GetHubStats()
	reqID, ok := r.Context().Value(middleware.RequestIdKey).(string)
	if !ok {
		reqID = "unknown"
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(handlers.CreateResponse("get websocket stats", stats, reqID))
	return nil
}

// Room handlers

func (h *HubHandler) HandleGetRoomStats(w http.ResponseWriter, r *http.Request) *app_error.AppError {
	roomID := chi.URLParam(r, "roomId")
	stats := h.Hub.GetRoomStats(roomID)
	reqID, ok := r.Context().Value(middleware.RequestIdKey).(string)
	if !ok {
		reqID = "unknown"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(handlers.CreateResponse("get websocket room stats", stats, reqID))

	return nil
}

func (h *HubHandler) HandleGetRoomClients(w http.ResponseWriter, r *http.Request) *app_error.AppError {
	roomID := chi.URLParam(r, "roomId")
	clients := h.Hub.GetRoomClients(roomID)

	type ClientInfo struct {
		ID          string    `json:"id"`
		UserID      string    `json:"user_id"`
		ConnectedAt time.Time `json:"connected_at"`
		LastSeen    time.Time `json:"last_seen"`
	}

	var clientList []ClientInfo
	for _, client := range clients {
		clientList = append(clientList, ClientInfo{
			ID:          client.ID,
			UserID:      client.UserID,
			ConnectedAt: client.ConnectedAt,
			LastSeen:    client.GetLastSeen(),
		})
	}

	resp := map[string]any{
		"room_id": roomID,
		"count":   len(clientList),
		"clients": clientList,
	}
	reqID, ok := r.Context().Value(middleware.RequestIdKey).(string)
	if !ok {
		reqID = "unknown"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(handlers.CreateResponse("successfully get rooms client", resp, reqID))
	return nil
}

func (h *HubHandler) HandleBroadcastToRoom(w http.ResponseWriter, r *http.Request) *app_error.AppError {
	roomID := chi.URLParam(r, "roomID")

	var payload struct {
		Type     string         `json:"type"`
		Content  string         `json:"content"`
		Data     map[string]any `json:"data"`
		SenderID string         `json:"sender_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		return app_error.NewAppError(http.StatusBadRequest, "Invalid request body", "request-body-broadcast")
	}

	if payload.Content == "" {
		return app_error.NewAppError(http.StatusBadRequest, "Content is required", "payload-content-missing")
	}

	var msg websocket.OutgoingMessage
	if payload.Type == "system" || payload.Type == "" {
		msg = websocket.NewSystemMessage(roomID, payload.Content, payload.Data)
	} else {
		msg = websocket.OutgoingMessage{
			Type:      payload.Type,
			RoomID:    roomID,
			SenderID:  payload.SenderID,
			Data:      payload.Data,
			Timestamp: time.Now().Unix(),
		}
	}

	h.Hub.BroadcastToRoom(roomID, msg)
	reqID, ok := r.Context().Value(middleware.RequestIdKey).(string)
	if !ok {
		reqID = "unknown"
	}
	resp := map[string]any{
		"status":    "sent",
		"room_id":   roomID,
		"type":      payload.Type,
		"timestamp": time.Now().Unix(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(handlers.CreateResponse("successfully get rooms client", resp, reqID))
	return nil
}

// User handlers

func (h *HubHandler) HandleKickUser(w http.ResponseWriter, r *http.Request) *app_error.AppError {
	roomID := chi.URLParam(r, "roomId")

	var payload struct {
		UserID string `json:"user_id"`
		Reason string `json:"reason"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		return app_error.NewAppError(http.StatusBadRequest, "invalid request body", "request-body-kick-user")
	}

	clients := h.Hub.GetRoomClients(roomID)
	kicked := 0

	for _, client := range clients {
		if client.UserID == payload.UserID {
			kickMsg := websocket.NewSystemMessage(roomID, fmt.Sprintf("You have been removed from the room. Reason: %s", payload.Reason), map[string]any{"action": "kicaked"})
			client.SendMessage(kickMsg)
			client.Close()
			kicked++
		}
	}

	resp := map[string]any{
		"status":         "success",
		"kicked_clients": kicked,
		"user_id":        payload.UserID,
	}

	reqID, ok := r.Context().Value(middleware.RequestIdKey).(string)
	if !ok {
		reqID = "unknown"
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(handlers.CreateResponse("successfully kick users", resp, reqID))

	return nil
}

func (h *HubHandler) HandleGetUserStatus(w http.ResponseWriter, r *http.Request) *app_error.AppError {
	userID := chi.URLParam(r, "userId")
	roomID := r.URL.Query().Get("roomId")

	var isOnline bool
	var activeClients int

	if roomID == "" {
		isOnline = h.Hub.IsUserOnlineInRoom(roomID, userID)
	} else {
		clients := h.Hub.GetUserClients(userID)
		activeClients = len(clients)
		isOnline = activeClients > 0
	}

	resp := map[string]any{
		"user_id":        userID,
		"online":         isOnline,
		"active_clients": activeClients,
		"room_id":        roomID,
	}

	reqID, ok := r.Context().Value(middleware.RequestIdKey).(string)
	if !ok {
		reqID = "unknown"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(handlers.CreateResponse("successful get user status", resp, reqID))

	return nil
}

func (h *HubHandler) HandleGetUserConnections(w http.ResponseWriter, r *http.Request) *app_error.AppError {
	userID := chi.URLParam(r, "userId")
	clients := h.Hub.GetUserClients(userID)

	type ConnectionInfo struct {
		ClientID    string    `json:"client_id"`
		RoomID      string    `json:"room_id"`
		ConnectedAt time.Time `json:"connected_at"`
		LastSeen    time.Time `json:"last_seen"`
		IsActive    bool      `json:"is_active"`
	}

	var connections []ConnectionInfo
	for _, client := range clients {
		connections = append(connections, ConnectionInfo{
			ClientID:    client.ID,
			RoomID:      client.RoomID,
			ConnectedAt: client.ConnectedAt,
			LastSeen:    client.GetLastSeen(),
			IsActive:    client.IsClientActive(),
		})
	}

	resp := map[string]any{
		"user_id":     userID,
		"count":       len(connections),
		"connections": connections,
	}

	reqID, ok := r.Context().Value(middleware.RequestIdKey).(string)
	if !ok {
		reqID = "unknown"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(handlers.CreateResponse("successfully get user connection", resp, reqID))

	return nil
}

func (h *HubHandler) HandleBroadcastToUser(w http.ResponseWriter, r *http.Request) *app_error.AppError {
	userID := chi.URLParam(r, "userId")

	var payload struct {
		Type    string         `json:"type"`
		Content string         `json:"content"`
		Data    map[string]any `json:"data"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		return app_error.NewAppError(http.StatusBadRequest, "invalid request body", "request-body-broadcast-user")
	}

	msg := websocket.OutgoingMessage{
		Type:      payload.Type,
		Data:      payload.Data,
		Timestamp: time.Now().Unix(),
	}

	h.Hub.BroadcastToUser(userID, msg)

	resp := map[string]any{
		"status":    "sent",
		"user_id":   userID,
		"type":      payload.Type,
		"timestamp": time.Now().Unix(),
	}
	reqID, ok := r.Context().Value(middleware.RequestIdKey).(string)
	if !ok {
		reqID = "unknown"
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(handlers.CreateResponse("successfully broadcast to user", resp, reqID))

	return nil
}

func (h *HubHandler) HandleDisconnectUser(w http.ResponseWriter, r *http.Request) *app_error.AppError {
	userID := chi.URLParam(r, "userId")

	var payload struct {
		Reason string `json:"reason"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		return app_error.NewAppError(http.StatusBadRequest, "invalid request body", "request-body-disconnect-user")
	}

	clients := h.Hub.GetUserClients(userID)
	disconnected := 0

	for _, client := range clients {
		msg := websocket.NewSystemMessage(client.RoomID, fmt.Sprintf("Connection closed: %s", payload.Reason), map[string]any{"action": "force_disconnect"})
		client.SendMessage(msg)
		client.Close()
		disconnected++
	}

	resp := map[string]any{
		"status":               "success",
		"disconnected_clients": disconnected,
		"user_id":              userID,
		"reason":               payload.Reason,
	}

	reqID, ok := r.Context().Value(middleware.RequestIdKey).(string)
	if !ok {
		reqID = "unknown"
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(handlers.CreateResponse("successfully disconnect user", resp, reqID))

	return nil
}
