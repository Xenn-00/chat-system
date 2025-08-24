package websocket

import (
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// TODO: limit your cors, don't get true so easy in production
	CheckOrigin: func(r *http.Request) bool { return true },
}

// call this from router chi

func (h *Hub) HandleWS(w http.ResponseWriter, r *http.Request, userID, roomID string) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error().Err(err).Msg("ws: upgrade failed")
		http.Error(w, "upgrade failed", http.StatusBadRequest)
		return
	}

	client := &Client{
		ID:     userID,
		RoomID: roomID,
		Conn:   conn,
		Send:   make(chan []byte, 256), // buffer just as good as you go
	}

	h.Register(roomID, client)
}
