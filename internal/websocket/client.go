package websocket

import (
	"encoding/json"

	gorilla "github.com/gorilla/websocket"
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

func (c *Client) WritePump() {
	defer c.Conn.Close()

	for message := range c.Send {
		if err := c.Conn.WriteMessage(gorilla.TextMessage, message); err != nil {
			return
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
