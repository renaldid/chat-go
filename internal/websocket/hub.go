package websocket

import (
	"encoding/json"
	"sync"
)

type Event struct {
	Type  string `json:"type"`
	Data  any    `json:"data,omitempty"`
	Error string `json:"error,omitempty"`
}

type Hub struct {
	mu    sync.RWMutex
	rooms map[int64]map[*Client]struct{}
}

func NewHub() *Hub {
	return &Hub{
		rooms: make(map[int64]map[*Client]struct{}),
	}
}

func (h *Hub) Register(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, exists := h.rooms[client.ConversationID]; !exists {
		h.rooms[client.ConversationID] = make(map[*Client]struct{})
	}

	h.rooms[client.ConversationID][client] = struct{}{}
}

func (h *Hub) Unregister(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	room, exists := h.rooms[client.ConversationID]
	if !exists {
		return
	}

	if _, exists := room[client]; !exists {
		return
	}

	delete(room, client)
	close(client.Send)

	if len(room) == 0 {
		delete(h.rooms, client.ConversationID)
	}
}

func (h *Hub) Broadcast(conversationID int64, event Event) {
	payload, err := json.Marshal(event)
	if err != nil {
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	room, exists := h.rooms[conversationID]
	if !exists {
		return
	}

	for client := range room {
		select {
		case client.Send <- payload:
		default:
		}
	}
}
