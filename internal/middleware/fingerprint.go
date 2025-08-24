package middleware

import (
	"context"
	"net/http"

	app_error "github.com/xenn00/chat-system/internal/errors"
)

type fingerprintKey string

const FingerprintKey fingerprintKey = "deviceFingerprint"

func GetDeviceFingerprint(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fingerprint := r.Header.Get("X-Device-Fingerprint")
		if fingerprint == "" {
			writeAppError(w, app_error.NewAppError(http.StatusBadRequest, "Missing device fingerprint", "fingerprint"))
			return
		}
		ctx := context.WithValue(r.Context(), FingerprintKey, fingerprint)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}
