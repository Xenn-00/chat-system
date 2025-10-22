package routers

import (
	"github.com/go-chi/chi/v5"
	"github.com/xenn00/chat-system/internal/handlers"
	hub_handler "github.com/xenn00/chat-system/internal/handlers/hub-handler"
	"github.com/xenn00/chat-system/internal/websocket"
)

func HubRouter(r chi.Router, wsHub *websocket.Hub) {
	hubHandler := hub_handler.NewHubHandler(wsHub)
	r.Route("/api/v1", func(r chi.Router) {
		// Health stats
		r.Get("/health", hubHandler.HandleHealth)
		r.Get("/stats", handlers.WrapHandler(hubHandler.HandleGetStats))

		// Room routes
		r.Route("/rooms/{roomId}", func(r chi.Router) {
			r.Get("/stats", handlers.WrapHandler(hubHandler.HandleGetRoomStats))
			r.Get("/clients", handlers.WrapHandler(hubHandler.HandleBroadcastToRoom))
			r.Post("/broadcast", handlers.WrapHandler(hubHandler.HandleBroadcastToRoom))
			r.Post("/kick", handlers.WrapHandler(hubHandler.HandleKickUser))
		})

		// User routes
		r.Route("/users/{userId}", func(r chi.Router) {
			r.Get("/status", handlers.WrapHandler(hubHandler.HandleGetUserStatus))
			r.Get("/connections", handlers.WrapHandler(hubHandler.HandleGetUserConnections))
			r.Post("/broadcast", handlers.WrapHandler(hubHandler.HandleBroadcastToUser))
			r.Post("/disconnect", handlers.WrapHandler(hubHandler.HandleDisconnectUser))
		})
	})
}
