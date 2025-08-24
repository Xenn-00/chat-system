package routers

import (
	"github.com/go-chi/chi/v5"
	"github.com/xenn00/chat-system/internal/handlers"
	"github.com/xenn00/chat-system/internal/middleware"
	"github.com/xenn00/chat-system/state"
)

func ChatRouter(r chi.Router, state *state.AppState) {
	chatHandler := handlers.NewChatHandler(state)
	r.Group(func(protected chi.Router) {
		protected.Use(middleware.JWTAuthWithAutoRefresh(state.JwtSecret.Private, state.JwtSecret.Public, state.Redis))
		protected.Post("/api/v1/chat/{receiverId}/messages", handlers.WrapHandler(chatHandler.SendPrivateMessage))
	})
}
