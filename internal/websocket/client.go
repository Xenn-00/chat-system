package websocket

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 1 << 20 // 1 MB
	channelBuffer  = 256
)

type Client struct {
	// Client identification
	ID     string `json:"id"`
	UserID string `json:"user_id"`
	RoomID string `json:"room_id"`

	// Websocket connection
	Conn *websocket.Conn `json:"-"`

	// Communication channels
	Send    chan []byte           `json:"-"`
	Receive chan *IncomingMessage `json:"-"`

	// Client state
	Hub         *Hub      `json:"-"`
	IsActive    bool      `json:"is_active"`
	ConnectedAt time.Time `json:"connected_at"`
	LastSeen    time.Time `json:"last_seen"`

	// Concurrency control
	mu     sync.RWMutex
	ctx    context.Context
	cancel context.CancelFunc
}

type IncomingMessage struct {
	Type      string          `json:"type"`
	Data      json.RawMessage `json:"data"`
	Timestamp int64           `json:"timestamp"`
	ClientID  string          `json:"client_id"`
}

func NewClient(userID, roomID string, conn *websocket.Conn, hub *Hub) *Client {
	ctx, cancel := context.WithCancel(context.Background())
	client := &Client{
		ID:          generateClientID(userID),
		UserID:      userID,
		RoomID:      roomID,
		Conn:        conn,
		Send:        make(chan []byte, channelBuffer),
		Receive:     make(chan *IncomingMessage, channelBuffer),
		Hub:         hub,
		IsActive:    true,
		ConnectedAt: time.Now(),
		LastSeen:    time.Now(),
		ctx:         ctx,
		cancel:      cancel,
	}

	return client
}

func (c *Client) Start() {
	go c.writePump()
	go c.readPump()
	go c.messagePump()
}

// writePump: take data from c.Send and send to socket + ping
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.cleanup()
	}()

	for {
		select {
		case <-c.ctx.Done():
			return
		case msg, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// close channel
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				log.Error().Err(err).Str("clientID", c.ID).Msg("ws: failed to get writer")
				return
			}

			// Write main message
			if _, err := w.Write(msg); err != nil {
				log.Error().Err(err).Str("clientID", c.ID).Msg("ws: failed to write message")
				w.Close()
				return
			}

			// Batch additional messages if available (performance optimization)
			n := len(c.Send)
		batchLoop:
			for range n {
				select {
				case msg := <-c.Send:
					if _, err := w.Write([]byte("\n")); err != nil {
						log.Error().Err(err).Msg("failed to write separator")
						w.Close()
						return
					}
					if _, err := w.Write(msg); err != nil {
						log.Error().Err(err).Msg("failed to write batched message")
						w.Close()
						return
					}
				default:
					break batchLoop
				}
			}

			if err := w.Close(); err != nil {
				log.Error().Err(err).Str("clientID", c.ID).Msg("ws: failed to close writer")
				return
			}

			c.updateLastSeen()

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Error().Err(err).Str("clientID", c.ID).Msg("ws: failed to send ping")
				return
			}
		}
	}
}

// readPump handles incoming messages from client
func (c *Client) readPump() {
	defer c.cleanup()

	c.Conn.SetReadLimit(maxMessageSize)
	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		c.updateLastSeen()
		return nil
	})
	for {
		select {
		case <-c.ctx.Done():
			return
		default:
		}

		var incomingMsg IncomingMessage
		err := c.Conn.ReadJSON(&incomingMsg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Error().Err(err).Str("clientID", c.ID).Msg("ws: unexpected close error")
			}
			break
		}

		// Add metadata
		incomingMsg.Timestamp = time.Now().Unix()
		incomingMsg.ClientID = c.ID

		// Send to message processor
		select {
		case c.Receive <- &incomingMsg:
		default:
			log.Warn().Str("clientID", c.ID).Msg("ws: message receive buffer full, dropping message")
		}

		c.updateLastSeen()
	}
}

// messagePump processes incoming messages
func (c *Client) messagePump() {
	for {
		select {
		case <-c.ctx.Done():
			return
		case msg := <-c.Receive:
			c.handleIncomingMessage(msg)
		}
	}
}

// handleIncomingMessage routes incoming messages to appropriate handlers
func (c *Client) handleIncomingMessage(msg *IncomingMessage) {
	switch msg.Type {
	case "join_room":
		c.handleJoinRoom(msg.Data)
	case "leave_room":
		c.handleLeaveRoom(msg.Data)
	case "typing_start":
		c.handleTypingStart(msg.Data)
	case "typing_stop":
		c.handleTypingStop(msg.Data)
	case "ping":
		c.handlePing()
	default:
		log.Warn().Str("clientID", c.ID).Str("messageType", msg.Type).Msg("ws: unknown message type")
	}
}

// Message handlers
func (c *Client) handleJoinRoom(data json.RawMessage) {
	var joinData struct {
		RoomID string `json:"room_id"`
	}

	if err := json.Unmarshal(data, &joinData); err != nil {
		log.Error().Err(err).Str("clientID", c.ID).Msg("ws: invalid join room data")
		return
	}

	// Update client room
	oldRoomID := c.RoomID
	c.mu.Lock()
	c.RoomID = joinData.RoomID
	c.mu.Unlock()

	// Unregister from old room, register to new room
	if oldRoomID != "" && oldRoomID != joinData.RoomID {
		c.Hub.Unregister(oldRoomID, c)
	}

	c.Hub.Register(joinData.RoomID, c)

	// Send confirmation
	response := OutgoingMessage{
		Type:   "room_joined",
		RoomID: joinData.RoomID,
		Data: map[string]any{
			"room_id": joinData.RoomID,
			"user_id": c.UserID,
		},
		Timestamp: time.Now().Unix(),
	}
	c.SendMessage(response)

	log.Info().Str("clientID", c.ID).Str("roomID", joinData.RoomID).Msg("ws: client joined room")
}

func (c *Client) handleLeaveRoom(data json.RawMessage) {
	c.Hub.Unregister(c.RoomID, c)

	response := OutgoingMessage{
		Type:   "room_left",
		RoomID: c.RoomID,
		Data: map[string]any{
			"room_id": c.RoomID,
			"user_id": c.UserID,
		},
		Timestamp: time.Now().Unix(),
	}
	c.SendMessage(response)

	log.Info().Str("clientID", c.ID).Str("roomID", c.RoomID).Msg("ws: client left room")
}

func (c *Client) handleTypingStart(data json.RawMessage) {
	c.Hub.BroadcastToRoomExcept(c.RoomID, OutgoingMessage{
		Type:     "user_typing",
		RoomID:   c.RoomID,
		SenderID: c.UserID,
		Data: map[string]interface{}{
			"user_id": c.UserID,
			"typing":  true,
		},
		Timestamp: time.Now().Unix(),
	}, c)
}

func (c *Client) handleTypingStop(data json.RawMessage) {
	c.Hub.BroadcastToRoomExcept(c.RoomID, OutgoingMessage{
		Type:     "user_typing",
		RoomID:   c.RoomID,
		SenderID: c.UserID,
		Data: map[string]interface{}{
			"user_id": c.UserID,
			"typing":  false,
		},
		Timestamp: time.Now().Unix(),
	}, c)
}

func (c *Client) handlePing() {
	response := OutgoingMessage{
		Type:      "pong",
		Timestamp: time.Now().Unix(),
	}
	c.SendMessage(response)
}

func (c *Client) SendMessage(msg OutgoingMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Error().Err(err).Str("clientID", c.ID).Msg("ws: failed to marshal message")
		return
	}

	select {
	case c.Send <- data:
	case <-c.ctx.Done():
	default:
		log.Warn().Str("clientID", c.ID).Msg("ws: send buffer full, dropping message")
		go c.Close()
	}
}

// Close gracefully closes the client connection
func (c *Client) Close() {
	c.mu.Lock()
	if !c.IsActive {
		c.mu.Unlock()
		return
	}

	c.IsActive = false
	c.mu.Unlock()

	c.cancel()
	c.cleanup()
}

// cleanup handles client cleanup
func (c *Client) cleanup() {
	// Unregister from hub
	if c.Hub != nil && c.RoomID != "" {
		c.Hub.Unregister(c.RoomID, c)
	}

	// Close channels
	select {
	case <-c.Send:
	default:
		close(c.Send)
	}

	select {
	case <-c.Receive:
	default:
		close(c.Receive)
	}

	// Close connection
	if c.Conn != nil {
		c.Conn.Close()
	}

	log.Info().Str("clientID", c.ID).Str("roomID", c.RoomID).Dur("duration", time.Since(c.ConnectedAt)).Msg("ws: client disconnected")
}

func (c *Client) updateLastSeen() {
	c.mu.Lock()
	c.LastSeen = time.Now()
	c.mu.Unlock()
}

func (c *Client) GetLastSeen() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.LastSeen
}

func (c *Client) IsClientActive() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.IsActive
}

// generateClientID creates a unique client ID
func generateClientID(userID string) string {
	return userID + "_" + time.Now().Format("20060102150405") + "_" + generateRandomString(6)
}

func generateRandomString(length int) string {
	// Simple random string generation (you might want to use crypto/rand for production)
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}
