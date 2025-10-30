package websocket

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

type Hub struct {
	// Room management
	rooms map[string]map[*Client]struct{}
	mu    sync.RWMutex

	// User tracking
	userClients map[string][]*Client // userID -> [clients]
	userMu      sync.RWMutex

	// Hub lifecycle
	ctx    context.Context
	cancel context.CancelFunc

	// Metrics
	stats   HubStats
	statsMu sync.RWMutex

	// Cleanup
	cleanupTicker *time.Ticker
}

type HubStats struct {
	TotalRooms       int       `json:"total_rooms"`
	TotalClients     int       `json:"total_clients"`
	TotalConnections int64     `json:"total_connections"`
	MessageSent      int64     `json:"message_sent"`
	LastReset        time.Time `json:"last_reset"`
}

func NewHub() *Hub {
	ctx, cancel := context.WithCancel(context.Background())
	hub := &Hub{
		rooms:       make(map[string]map[*Client]struct{}),
		userClients: make(map[string][]*Client),
		ctx:         ctx,
		cancel:      cancel,
		stats: HubStats{
			LastReset: time.Now(),
		},
		cleanupTicker: time.NewTicker(1 * time.Minute),
	}

	// Start cleanup routine
	go hub.cleanupRoutine()

	return hub
}

// Register adds a client to a room
func (h *Hub) Register(roomId string, client *Client) {
	h.mu.Lock()

	// Initialize room if doesn't exist
	if h.rooms[roomId] == nil {
		h.rooms[roomId] = make(map[*Client]struct{})
	}

	// Add client to room
	h.rooms[roomId][client] = struct{}{}
	h.mu.Unlock()

	// track user clients
	h.userMu.Lock()
	h.userClients[client.UserID] = append(h.userClients[client.UserID], client)
	h.userMu.Unlock()

	// Update stats
	h.updateStats(func(stats *HubStats) {
		stats.TotalConnections++
	})

	// Start client pumps
	client.Start()

	// Broadcast user online status
	h.broadcastUserStatus(roomId, client.UserID, true)

	log.Info().Str("roomID", roomId).Str("clientID", client.ID).Str("userID", client.UserID).Int("roomSize", len(h.rooms[roomId])).Msg("ws: client registered to room")
}

// Unregister removes a client from a room
func (h *Hub) Unregister(roomId string, client *Client) {
	h.mu.Lock()
	if clients, ok := h.rooms[roomId]; ok {
		delete(clients, client)

		// Clean up empty rooms
		if len(clients) == 0 {
			delete(h.rooms, roomId)
		}
	}
	h.mu.Unlock()

	// Remove from user clients tracking
	h.userMu.Lock()
	userClients := h.userClients[client.UserID]
	for i, c := range userClients {
		if c == client {
			// Remove client from slice
			h.userClients[client.UserID] = append(h.userClients[client.UserID], userClients[i+1:]...)
			break
		}
	}

	// Clean up empty user entries
	if len(h.userClients[client.UserID]) == 0 {
		delete(h.userClients, client.UserID)
	}

	h.userMu.Unlock()

	// Check if user is still online in this room
	isUserStillOnline := h.IsUserOnlineInRoom(roomId, client.UserID)
	if !isUserStillOnline {
		// Broadcast user offline status
		h.broadcastUserStatus(roomId, client.UserID, false)
	}

	log.Info().Str("roomID", roomId).Str("clientID", client.ID).Str("userID", client.UserID).Msg("ws: client unregistered from room")
}

// BroadcastToRoom sends a message to all clients in a room
func (h *Hub) BroadcastToRoom(roomId string, message OutgoingMessage) {
	h.broadcastToRoomInternal(roomId, message, nil)
}

// BroadcastToRoomExcept sends a message to all clients in a room except the specified client
func (h *Hub) BroadcastToRoomExcept(roomId string, message OutgoingMessage, except *Client) {
	h.broadcastToRoomInternal(roomId, message, except)
}

// Internal broadcast logic
func (h *Hub) broadcastToRoomInternal(roomID string, message OutgoingMessage, except *Client) {
	// Set room ID in message
	message.RoomID = roomID

	data, err := json.Marshal(message)
	if err != nil {
		log.Error().Err(err).Str("roomID", roomID).Msg("ws: failed to marshal broadcast message")
		return
	}

	// Get snapshot of clients (minimize lock time)
	h.mu.RLock()
	var targets []*Client
	if clients, ok := h.rooms[roomID]; ok {
		targets = make([]*Client, 0, len(clients))
		for client := range clients {
			if except != nil && client == except {
				continue
			}
			if client.IsClientActive() {
				targets = append(targets, client)
			}
		}
	}
	h.mu.RUnlock()

	if len(targets) == 0 {
		return
	}

	// Send to clients outside of lock (parallel sending)
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 50) // limit concurrent sends

	for _, client := range targets {
		wg.Add(1)
		go func(c *Client) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() {
				<-semaphore
			}()
			select {
			case c.Send <- data:
				// success
			case <-c.ctx.Done():
				// Client is closing
			default:
				// Client buffer full - slow consumer
				log.Warn().Str("roomID", roomID).Str("clientID", c.ID).Msg("ws: slow consumer, dropping message")

				// Auto-cleanup slow clients
				go c.Close()
			}
		}(client)
	}

	wg.Wait()

	// Update stats
	h.updateStats(func(stats *HubStats) {
		stats.MessageSent += int64(len(targets))
	})

	log.Debug().Str("roomID", roomID).Int("targets", len(targets)).Str("messageType", message.Type).Msg("ws: broadcast completed")
}

// BroadcastToUser sends a message to all connections of a specific user
func (h *Hub) BroadcastToUser(userID string, message OutgoingMessage) {
	h.userMu.RLock()
	clients := make([]*Client, len(h.userClients[userID]))
	copy(clients, h.userClients[userID])
	h.userMu.RUnlock()

	if len(clients) == 0 {
		return
	}

	data, err := json.Marshal(message)
	if err != nil {
		log.Error().Err(err).Str("userID", userID).Msg("ws: failed to marshal user message")
		return
	}

	for _, client := range clients {
		if !client.IsClientActive() {
			continue
		}

		select {
		case client.Send <- data:
		case <-client.ctx.Done():
		default:
			log.Warn().Str("userID", userID).Str("clientID", client.ID).Msg("ws: user client buffer full")
		}
	}
}

// Utility methods

// GetRoomClients return all active clients in a room
func (h *Hub) GetRoomClients(roomID string) []*Client {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var clients []*Client
	if roomClients, ok := h.rooms[roomID]; ok {
		for client := range roomClients {
			if client.IsClientActive() {
				clients = append(clients, client)
			}
		}
	}

	return clients
}

// GetUserClients returns all active clients for a user
func (h *Hub) GetUserClients(userID string) []*Client {
	h.userMu.RLock()
	defer h.userMu.RUnlock()

	var activeClients []*Client
	for _, client := range h.userClients[userID] {
		if client.IsClientActive() {
			activeClients = append(activeClients, client)
		}
	}

	return activeClients
}

// IsUserOnlineInRoom checks if a user has any active connections in a room
func (h *Hub) IsUserOnlineInRoom(roomID, userID string) bool {
	h.mu.RLock()
	roomClients, ok := h.rooms[roomID]
	h.mu.RUnlock()

	if !ok {
		return false
	}

	for client := range roomClients {
		if client.UserID == userID && client.IsClientActive() {
			return true
		}
	}

	return false
}

// GetRoomStats returns statistics for a room
func (h *Hub) GetRoomStats(roomID string) map[string]interface{} {
	h.mu.RLock()
	defer h.mu.RUnlock()

	stats := map[string]interface{}{
		"room_id": roomID,
		"exists":  false,
	}

	if clients, ok := h.rooms[roomID]; ok {
		activeClients := 0
		uniqueUsers := make(map[string]bool)

		for client := range clients {
			if client.IsClientActive() {
				activeClients++
				uniqueUsers[client.UserID] = true
			}
		}

		stats["exists"] = true
		stats["total_connections"] = len(clients)
		stats["active_connections"] = activeClients
		stats["unique_users"] = len(uniqueUsers)
	}

	return stats
}

// GetHubStats returns overall hub statistics
func (h *Hub) GetHubStats() HubStats {
	h.statsMu.RLock()
	defer h.statsMu.RUnlock()

	// Update current counts
	h.mu.RLock()
	h.stats.TotalRooms = len(h.rooms)

	totalClients := 0
	for _, clients := range h.rooms {
		for client := range clients {
			if client.IsClientActive() {
				totalClients++
			}
		}
	}
	h.stats.TotalClients = totalClients
	h.mu.RUnlock()

	return h.stats
}

func (h *Hub) broadcastUserStatus(roomID, userID string, online bool) {
	status := "offline"
	if online {
		status = "online"
	}

	message := OutgoingMessage{
		Type:   "user_status",
		RoomID: roomID,
		Data: map[string]any{
			"user_id": userID,
			"status":  status,
		},
		Timestamp: time.Now().Unix(),
	}

	// Don't broadcast to the user themselves
	h.broadcaseToRoomExceptUser(roomID, message, userID)
}

func (h *Hub) broadcaseToRoomExceptUser(roomID string, message OutgoingMessage, exceptUserID string) {
	data, err := json.Marshal(message)
	if err != nil {
		return
	}

	h.mu.RLock()
	var targets []*Client
	if clients, ok := h.rooms[roomID]; ok {
		for client := range clients {
			if client.UserID != exceptUserID && client.IsClientActive() {
				targets = append(targets, client)
			}
		}
	}
	h.mu.RUnlock()

	for _, client := range targets {
		select {
		case client.Send <- data:
		default:
		}
	}
}

func (h *Hub) updateStats(fn func(*HubStats)) {
	h.statsMu.Lock()
	fn(&h.stats)
	h.statsMu.Unlock()
}

func (h *Hub) cleanupRoutine() {
	defer h.cleanupTicker.Stop()

	for {
		select {
		case <-h.ctx.Done():
			return
		case <-h.cleanupTicker.C:
			h.performCleanup()
		}
	}
}

func (h *Hub) performCleanup() {
	now := time.Now()
	inactiveThreshold := 2 * time.Minute

	// Clean up inactive clients
	var toRemove []*Client

	h.mu.RLock()
	for _, clients := range h.rooms {
		for client := range clients {
			if !client.IsClientActive() || now.Sub(client.GetLastSeen()) > inactiveThreshold {
				toRemove = append(toRemove, client)
			}
		}
	}
	h.mu.RUnlock()

	for _, client := range toRemove {
		log.Info().
			Str("clientID", client.ID).
			Str("roomID", client.RoomID).
			Msg("ws: cleaning up inactive client")
		client.Close()
	}

	log.Debug().Int("cleaned", len(toRemove)).Msg("ws: cleanup routine completed")
}

// Close gracefully shuts down the hub
func (h *Hub) Close() {
	log.Info().Msg("ws: shutting down hub")

	h.cancel()

	// Close all clients
	h.mu.RLock()
	var allClients []*Client
	for _, clients := range h.rooms {
		for client := range clients {
			allClients = append(allClients, client)
		}
	}
	h.mu.RUnlock()

	for _, client := range allClients {
		client.Close()
	}

	log.Info().Int("clients", len(allClients)).Msg("ws: hub shutdown completed")
}
