package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

type requestIdKey string

const RequestIdKey requestIdKey = "requestId"

func WithRequestId(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqId := uuid.New().String()
		ctx := context.WithValue(r.Context(), RequestIdKey, reqId)
		r = r.WithContext(ctx)
		r.Header.Set("X-Request-ID", reqId)

		next.ServeHTTP(w, r)
	})
}
