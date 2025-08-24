package websocket

import (
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 1 << 20 // 1 MB
)

type Client struct {
	ID     string
	Conn   *websocket.Conn
	Send   chan []byte
	RoomID string
}

// writePump: take data from c.Send and send to socket + ping
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		_ = c.Conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.Send:
			_ = c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// close channel
				_ = c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}

			if _, err := w.Write(msg); err != nil {
				_ = w.Close()
				return
			}

			_ = w.Close()

		case <-ticker.C:
			_ = c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// readPump: read from client (if needed) + handle pong for keep-alive
func (c *Client) readPump(h *Hub) {
	defer func() {
		h.Unregister(c.RoomID, c)
		close(c.Send)
		_ = c.Conn.Close()
	}()

	c.Conn.SetReadLimit(maxMessageSize)
	_ = c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error {
		_ = c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		// if we doesn't need inbound WS (client -> server), we can discard it
		// _, _, err := c.Conn.ReadMessage()
		// if err != nil {break}

		// inbound
		if _, _, err := c.Conn.ReadMessage(); err != nil {
			break

		}
	}
}
