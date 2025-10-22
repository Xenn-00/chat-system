package routers

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	local_middleware "github.com/xenn00/chat-system/internal/middleware"
	"github.com/xenn00/chat-system/internal/websocket"
	"github.com/xenn00/chat-system/state"
)

func NewRouter(state *state.AppState, wsHub *websocket.Hub, wsHandler *websocket.WebSocketHandler) http.Handler {
	r := chi.NewRouter()
	r.Use(local_middleware.WithRequestId)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))
	r.Use(local_middleware.GetDeviceFingerprint)
	UserRouter(r, state)
	HubRouter(r, wsHub)
	ChatRouter(r, state)
	return r
}
