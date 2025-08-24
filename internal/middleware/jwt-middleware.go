package middleware

import (
	"context"
	"crypto/rsa"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
	app_error "github.com/xenn00/chat-system/internal/errors"
	"github.com/xenn00/chat-system/internal/utils"
	"github.com/xenn00/chat-system/internal/utils/types"
)

type claimsKey string

const UserClaimsKey claimsKey = "userClaims"

func JWTAuth(publicKey *rsa.PublicKey) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				writeAppError(w, app_error.NewAppError(http.StatusUnauthorized, "Missing Authorization header", "auth"))
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				writeAppError(w, app_error.NewAppError(http.StatusUnauthorized, "Invalid Authorization header format", "auth"))
				return
			}

			tokenStr := parts[1]

			claims, err := utils.ParseAndVerifySign(tokenStr, publicKey)
			if err != nil {
				log.Error().Err(err).Msg("jwt verify failed")
				writeAppError(w, app_error.NewAppError(http.StatusUnauthorized, "Invalid or expired token", "auth"))
				return
			}

			ctx := context.WithValue(r.Context(), UserClaimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func JWTAuthWithAutoRefresh(privateKey *rsa.PrivateKey, publicKey *rsa.PublicKey, redis *redis.Client) func(http.Handler) http.Handler {
	const (
		refreshTTL    = 7 * 24 * time.Hour
		statusValid   = "valid"
		statusRevoked = "revoked"
	)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fp, ok := r.Context().Value(FingerprintKey).(string)
			if !ok || fp == "" {
				writeAppError(w, app_error.NewAppError(http.StatusUnauthorized, "Missing device fingerprint", "fingerprint"))
				return
			}

			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				writeAppError(w, app_error.NewAppError(http.StatusUnauthorized, "Missing Authorization header", "auth"))
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				writeAppError(w, app_error.NewAppError(http.StatusUnauthorized, "Invalid Authorization header format", "auth"))
				return
			}

			tokenStr := parts[1]
			claims, err := utils.ParseAndVerifySign(tokenStr, publicKey)
			if err != nil {
				// expiry check
				if errors.Is(err, jwt.ErrTokenExpired) {
					refreshCookie, cerr := r.Cookie("refresh_token")
					if cerr != nil {
						writeAppError(w, app_error.NewAppError(http.StatusUnauthorized, "Refresh token missing", "auth"))
						return
					}

					refreshClaims, rErr := utils.ParseAndVerifySign(refreshCookie.Value, publicKey)
					if rErr != nil {
						writeAppError(w, app_error.NewAppError(http.StatusUnauthorized, "Invalid refresh token", "auth"))
						return
					}

					// check session in redis
					sessionKey := fmt.Sprintf("refresh:%s:%s:%s", refreshClaims.Sub, fp, *refreshClaims.Jti)

					session, err := utils.GetCacheData[types.RefreshSession](r.Context(), redis, sessionKey)
					if err != nil || session.Status != statusValid || session.ExpireAt < time.Now().Unix() {
						writeAppError(w, app_error.NewAppError(http.StatusUnauthorized, "Refresh token revoked or expired", "auth"))
						return
					}

					// generate new access + refresh
					newAccess, newRefresh, newJTI, genErr := utils.IssueNewTokens(refreshClaims.Sub, refreshClaims.Username, privateKey)
					if genErr != nil {
						writeAppError(w, app_error.NewAppError(http.StatusInternalServerError, "Failed to issue new tokens", "auth"))
						return
					}

					issue_at := time.Now().Unix()
					expires_refresh := issue_at + int64(refreshTTL.Seconds())

					newSessionKey := fmt.Sprintf("refresh:%s:%s:%s", refreshClaims.Sub, fp, newJTI)
					newSession := types.RefreshSession{
						UserId:      refreshClaims.Sub,
						JTI:         newJTI,
						Fingerprint: fp,
						IssueAt:     issue_at,
						ExpireAt:    expires_refresh,
						Status:      "valid",
					}

					utils.SetCacheData(r.Context(), redis, newSessionKey, &newSession, refreshTTL)

					// revoke old refresh
					session.Status = statusRevoked
					utils.SetCacheData(r.Context(), redis, sessionKey, &session, time.Duration(time.Until(time.Unix(session.ExpireAt, 0))))

					// set new refresh in cookie
					http.SetCookie(w, &http.Cookie{
						Name:     "refresh_token",
						Value:    newRefresh,
						HttpOnly: true,
						Secure:   true,
						SameSite: http.SameSiteStrictMode,
						Path:     "/",
						Expires:  time.Now().Add(7 * 24 * time.Hour),
					})

					// add new access token to header
					w.Header().Set("X-New-Access-Token", newAccess)
					claims, _ = utils.ParseAndVerifySign(newAccess, publicKey) // re-parsing
				} else {
					writeAppError(w, app_error.NewAppError(http.StatusUnauthorized, "Invalid token", "auth"))
					return
				}
			}
			sub := claims.Sub
			ctx := context.WithValue(r.Context(), UserClaimsKey, sub)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func writeAppError(w http.ResponseWriter, appErr *app_error.AppError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(appErr.Code)
	_ = appErr.JSON(w)
}
