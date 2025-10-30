package websocket

import (
	"context"
	"crypto/rsa"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
	"github.com/xenn00/chat-system/internal/middleware"
	"github.com/xenn00/chat-system/internal/utils"
)

func JWTWebSocketAuth(privateKey *rsa.PrivateKey, publicKey *rsa.PublicKey, redis *redis.Client) AuthenticatorFunc {
	return func(r *http.Request) (userID string, err error) {
		// 1. Get Fingerprint from context
		fp, ok := r.Context().Value(middleware.FingerprintKey).(string)
		if !ok || fp == "" {
			return "", &AuthError{Message: "missing device fingerprint"}
		}

		// 2. Try to get token
		token := getTokenFromRequest(r)

		// 3. Parse and verify token
		claims, err := utils.ParseAndVerifySign(token, publicKey)
		if err != nil {
			// If token is expired, try to refresh using cookie
			if errors.Is(err, jwt.ErrTokenExpired) {
				// For websocket, we can't refresh here because we can't set cookies in ws handshake
				// Client must refresh via HTTP endpoint first, then reconnect
				return "", &AuthError{Message: "token expired, please refresh and reconnect"}
			}
			return "", &AuthError{Message: "ïnvalid token"}
		}

		// 4. Optional: Validate session in Redis
		sessionKey := fmt.Sprintf("session:%s:%s", claims.Sub, fp)
		ctx := context.Background()

		exists, err := redis.Exists(ctx, sessionKey).Result()
		if err != nil || exists == 0 {
			return "", &AuthError{Message: "session not found or revoked"}
		}

		// 5. Return userID
		return claims.Sub, nil
	}
}

func getTokenFromRequest(r *http.Request) string {
	// Option 1: Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
			return parts[1]
		}
	}

	// Option 2: Query parameter
	token := r.URL.Query().Get("token")
	if token != "" {
		return token
	}

	// Option 3: Cookie
	cookie, err := r.Cookie("access_token")
	if err == nil && cookie.Value != "" {
		return cookie.Value
	}

	return ""
}
