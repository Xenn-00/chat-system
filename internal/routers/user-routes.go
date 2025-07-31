package routers

import (
	"github.com/go-chi/chi/v5"
	"github.com/xenn00/chat-system/internal/handlers"
	"github.com/xenn00/chat-system/state"
)

func UserRouter(r chi.Router, state *state.AppState) {
	userHandler := handlers.NewUserHandler(state)

	r.Post("/api/v1/users", handlers.WrapHandler(userHandler.CreateUser))
	r.Post("/api/v1/users/{userId}", handlers.WrapHandler(userHandler.VerifyUser))
}
