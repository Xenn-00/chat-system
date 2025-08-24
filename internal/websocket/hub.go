package websocket

import (
	"encoding/json"
	"sync"

	"github.com/rs/zerolog/log"
)

type Hub struct {
	rooms map[string]map[*Client]struct{}
	mu    sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		rooms: make(map[string]map[*Client]struct{}),
	}
}

func (h *Hub) Register(roomId string, client *Client) {
	h.mu.Lock()
	if h.rooms[roomId] == nil {
		h.rooms[roomId] = make(map[*Client]struct{})
	}
	h.rooms[roomId][client] = struct{}{}
	h.mu.Unlock()

	// start pumps
	go client.writePump()
	go client.readPump(h)
}

func (h *Hub) Unregister(roomId string, client *Client) {
	h.mu.Lock()
	if clients, ok := h.rooms[roomId]; ok {
		delete(clients, client)
		if len(clients) == 0 {
			delete(h.rooms, roomId)
		}
	}
	h.mu.Unlock()
}

// Nachrichtenübergabe, marshal zur JSON, broadcast ohne Halten der Sperre für längere Zeit
func (h *Hub) BroadcastToRoom(roomId string, payload Message) {
	data, err := json.Marshal(payload)
	if err != nil {
		log.Error().Err(err).Msg("ws: marshal payload")
		return
	}

	// snapshot client list under RLock, in order to not holding lock when sending
	h.mu.RLock()
	var targets []*Client
	if clients, ok := h.rooms[roomId]; ok {
		targets = make([]*Client, 0, len(clients))
		for c := range clients {
			targets = append(targets, c)
		}
	}
	h.mu.RUnlock()

	// sending outside lock; non-blocking so that slow consumer not depended into broadcast
	for _, c := range targets {
		select {
		case c.Send <- data:
		default:
			log.Warn().Str("roomId", roomId).Str("userId", c.ID).Msg("ws: slow consumer dropped")
			go func(cl *Client) {
				// clean up
				h.Unregister(c.RoomID, cl)
				close(cl.Send)
				_ = cl.Conn.Close()
			}(c)
		}
	}
}
