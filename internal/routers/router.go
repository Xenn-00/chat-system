package routers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/xenn00/chat-system/internal/middleware"
	"github.com/xenn00/chat-system/state"
)

func NewRouter(state *state.AppState) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.WithRequestId)
	UserRouter(r, state)
	return r
}
