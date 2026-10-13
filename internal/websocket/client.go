package websocket

import (
	"encoding/json"
	"log/slog"
	"time"

	gorilla "github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 64 * 1024
)

type Client struct {
	Conn           *gorilla.Conn
	ConversationID int64
	UserID         int64
	Send           chan []byte
}

func NewClient(conn *gorilla.Conn, conversationID int64, userID int64) *Client {
	return &Client{
		Conn:           conn,
		ConversationID: conversationID,
		UserID:         userID,
		Send:           make(chan []byte, 256),
	}
}

func (c *Client) ConfigureReadDeadline() {
	c.Conn.SetReadLimit(maxMessageSize)
	_ = c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(appData string) error {
		slog.Info(
			"websocket pong received",
			"user_id", c.UserID,
			"conversation_id", c.ConversationID,
		)

		return c.Conn.SetReadDeadline(
			time.Now().Add(pongWait),
		)
	})
}

func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		_ = c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			_ = c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.Conn.WriteMessage(gorilla.CloseMessage, []byte{})
				return
			}

			if err := c.Conn.WriteMessage(gorilla.TextMessage, message); err != nil {
				return
			}

		case <-ticker.C:
			_ = c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(gorilla.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *Client) SendEvent(event Event) {
	payload, err := json.Marshal(event)
	if err != nil {
		return
	}

	select {
	case c.Send <- payload:
	default:
	}
}
