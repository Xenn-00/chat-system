package websocket

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:    1024,
	WriteBufferSize:   1024,
	EnableCompression: true,

	// CORS configuration - customize for your needs
	CheckOrigin: func(r *http.Request) bool {
		// TODO: Implement proper CORS checking for production
		// For now, allowing all origins for development
		origin := r.Header.Get("Origin")

		// Allow local development
		if strings.Contains(origin, "localhost") || strings.Contains(origin, "127.0.0.1") {
			return true
		}

		// Add your production domains here
		allowedOrigins := []string{
			"https://yourdomain.com",
			"https://app.yourdomain.com",
		}

		for _, allowed := range allowedOrigins {
			if origin == allowed {
				return true
			}
		}

		// Reject unknown origins in production
		return true // Change to false in production
	},

	// Subprotocol negotiation (optional)
	Subprotocols: []string{"chat-v1"},
}

// WebSocketHandler handles WebSocket connections and upgrades
type WebSocketHandler struct {
	hub           *Hub
	rateLimiters  map[string]*RateLimiter
	rateLimiterMu sync.RWMutex
	authenticator AuthenticatorFunc

	// Configuration
	MaxConnections int
	RateLimit      RateLimitConfig

	Handler http.HandlerFunc
}

// AuthenticatorFunc validates WebSocket connections
type AuthenticatorFunc func(r *http.Request) (userID string, err error)

// RateLimitConfig configures rate limiting
type RateLimitConfig struct {
	Enabled           bool
	ConnectionsPerIP  int
	MessagesPerMinute int
	WindowSize        time.Duration
}

// RateLimiter tracks connection and message rates
type RateLimiter struct {
	connections map[string]int
	messages    map[string][]time.Time
	mu          sync.RWMutex
}

// NewWebSocketHandler creates a new WebSocket handler
func NewWebSocketHandler(hub *Hub, auth AuthenticatorFunc) *WebSocketHandler {
	h := &WebSocketHandler{
		hub:            hub,
		rateLimiters:   make(map[string]*RateLimiter),
		authenticator:  auth,
		MaxConnections: 1000, // Default max connections
		RateLimit: RateLimitConfig{
			Enabled:           true,
			ConnectionsPerIP:  10,
			MessagesPerMinute: 60,
			WindowSize:        time.Minute,
		},
	}

	h.Handler = h.HandleWebSocket
	return h
}

// HandleWebSocket handles WebSocket upgrade and connection management
func (h *WebSocketHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Extract room ID from URL or query parameters
	roomID := h.extractRoomID(r)
	if roomID == "" {
		log.Error().Msg("ws: room ID is required")
		http.Error(w, "Room ID is required", http.StatusBadRequest)
		return
	}

	// Authenticate user
	userID, err := h.authenticateConnection(r)
	if err != nil {
		log.Error().Err(err).Msg("ws: authentication failed")
		http.Error(w, "Authentication failed", http.StatusUnauthorized)
		return
	}

	// Rate limiting
	clientIP := h.getClientIP(r)
	if !h.checkRateLimit(clientIP, userID) {
		log.Warn().Str("ip", clientIP).Str("userID", userID).Msg("ws: rate limit exceeded")
		http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
		return
	}

	// Check max connections
	if h.hub.GetHubStats().TotalClients >= h.MaxConnections {
		log.Warn().Msg("ws: max connections reached")
		http.Error(w, "Server at capacity", http.StatusServiceUnavailable)
		return
	}

	// Upgrade connection
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error().Err(err).Msg("ws: failed to upgrade connection")
		return
	}

	// Create and register client
	client := NewClient(userID, roomID, conn, h.hub)

	// Set connection metadata
	client.Conn.SetReadLimit(maxMessageSize)

	// Log successful connection
	log.Info().
		Str("userID", userID).
		Str("roomID", roomID).
		Str("clientIP", clientIP).
		Str("clientID", client.ID).
		Msg("ws: new connection established")

	// Register client with hub
	h.hub.Register(roomID, client)

	// Update rate limiter
	h.updateConnectionCount(clientIP, 1)

	// Setup cleanup on disconnect
	go func() {
		<-client.ctx.Done()
		h.updateConnectionCount(clientIP, -1)
		log.Info().
			Str("clientID", client.ID).
			Str("userID", userID).
			Msg("ws: connection cleanup completed")
	}()
}

// Helper methods

func (h *WebSocketHandler) extractRoomID(r *http.Request) string {
	// Try URL path parameter first
	roomID := r.URL.Query().Get("room_id")
	if roomID != "" {
		return roomID
	}

	// Try from URL path
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) >= 2 && pathParts[len(pathParts)-2] == "rooms" {
		return pathParts[len(pathParts)-1]
	}

	return ""
}

func (h *WebSocketHandler) authenticateConnection(r *http.Request) (string, error) {
	if h.authenticator == nil {
		// Default authentication - extract from query param
		userID := r.URL.Query().Get("user_id")
		if userID == "" {
			return "", &AuthError{Message: "user_id is required"}
		}
		return userID, nil
	}

	return h.authenticator(r)
}

func (h *WebSocketHandler) getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	ip := r.RemoteAddr
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}

	return ip
}

func (h *WebSocketHandler) checkRateLimit(clientIP, userID string) bool {
	if !h.RateLimit.Enabled {
		return true
	}

	h.rateLimiterMu.RLock()
	limiter, exists := h.rateLimiters[clientIP]
	h.rateLimiterMu.RUnlock()

	if !exists {
		h.rateLimiterMu.Lock()
		limiter = &RateLimiter{
			connections: make(map[string]int),
			messages:    make(map[string][]time.Time),
		}
		h.rateLimiters[clientIP] = limiter
		h.rateLimiterMu.Unlock()
	}

	limiter.mu.RLock()
	connections := limiter.connections[clientIP]
	limiter.mu.RUnlock()

	return connections < h.RateLimit.ConnectionsPerIP
}

func (h *WebSocketHandler) updateConnectionCount(clientIP string, delta int) {
	h.rateLimiterMu.RLock()
	limiter, exists := h.rateLimiters[clientIP]
	h.rateLimiterMu.RUnlock()

	if !exists {
		return
	}

	limiter.mu.Lock()
	limiter.connections[clientIP] += delta
	if limiter.connections[clientIP] <= 0 {
		delete(limiter.connections, clientIP)
	}
	limiter.mu.Unlock()
}

// JWT-based authentication example
func JWTAuthenticator(secretKey []byte) AuthenticatorFunc {
	return func(r *http.Request) (string, error) {
		// Get token from query parameter or Authorization header
		token := r.URL.Query().Get("token")
		if token == "" {
			authHeader := r.Header.Get("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				token = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}

		if token == "" {
			return "", &AuthError{Message: "authentication token is required"}
		}

		// TODO: Implement JWT validation
		// This is a placeholder - implement your JWT validation logic
		// claims, err := validateJWT(token, secretKey)
		// if err != nil {
		//     return "", &AuthError{Message: "invalid token"}
		// }
		// return claims.UserID, nil

		// Placeholder implementation
		return "user_" + token[:8], nil // Use first 8 chars as userID
	}
}

// Session-based authentication example
func SessionAuthenticator(sessionStore SessionStore) AuthenticatorFunc {
	return func(r *http.Request) (string, error) {
		sessionID := r.URL.Query().Get("session_id")
		if sessionID == "" {
			// Try to get from cookies
			cookie, err := r.Cookie("session_id")
			if err != nil {
				return "", &AuthError{Message: "session_id is required"}
			}
			sessionID = cookie.Value
		}

		userID, err := sessionStore.GetUserID(sessionID)
		if err != nil {
			return "", &AuthError{Message: "invalid session"}
		}

		return userID, nil
	}
}

// Middleware for HTTP-based WebSocket upgrades

// WithRoomValidation validates room access before WebSocket upgrade
func (h *WebSocketHandler) WithRoomValidation(roomValidator RoomValidatorFunc) *WebSocketHandler {
	originalHandler := h.Handler

	h.Handler = func(w http.ResponseWriter, r *http.Request) {
		roomID := h.extractRoomID(r)
		userID, err := h.authenticateConnection(r)
		if err != nil {
			http.Error(w, "Authentication failed", http.StatusUnauthorized)
			return
		}

		if !roomValidator(roomID, userID) {
			log.Warn().
				Str("roomID", roomID).
				Str("userID", userID).
				Msg("ws: room access denied")
			http.Error(w, "Access denied", http.StatusForbidden)
			return
		}

		originalHandler(w, r)
	}

	return h
}

// WithLogging adds request logging
func (h *WebSocketHandler) WithLogging() *WebSocketHandler {
	originalHandler := h.Handler

	h.Handler = func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		log.Info().
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Str("remote_addr", r.RemoteAddr).
			Str("user_agent", r.Header.Get("User-Agent")).
			Msg("ws: connection attempt")

		originalHandler(w, r)

		log.Info().
			Dur("duration", time.Since(start)).
			Msg("ws: connection handling completed")
	}

	return h
}

// Cleanup routine for rate limiters
func (h *WebSocketHandler) StartCleanup(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			h.cleanupRateLimiters()
		}
	}
}

func (h *WebSocketHandler) cleanupRateLimiters() {
	now := time.Now()

	h.rateLimiterMu.Lock()
	defer h.rateLimiterMu.Unlock()

	for ip, limiter := range h.rateLimiters {
		limiter.mu.Lock()

		// Clean up old message timestamps
		for userID, timestamps := range limiter.messages {
			var validTimestamps []time.Time
			for _, ts := range timestamps {
				if now.Sub(ts) < h.RateLimit.WindowSize {
					validTimestamps = append(validTimestamps, ts)
				}
			}

			if len(validTimestamps) == 0 {
				delete(limiter.messages, userID)
			} else {
				limiter.messages[userID] = validTimestamps
			}
		}

		// Remove empty rate limiters
		if len(limiter.connections) == 0 && len(limiter.messages) == 0 {
			delete(h.rateLimiters, ip)
		}

		limiter.mu.Unlock()
	}
}

// Error types
type AuthError struct {
	Message string
}

func (e *AuthError) Error() string {
	return e.Message
}

// Interface types
type SessionStore interface {
	GetUserID(sessionID string) (string, error)
}

type RoomValidatorFunc func(roomID, userID string) bool
