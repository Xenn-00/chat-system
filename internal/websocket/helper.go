package websocket

import (
	"net/http"
	"strings"
	"time"
)

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
