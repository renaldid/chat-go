package websocket

import (
	"encoding/json"
	"sort"
	"sync"
)

type Event struct {
	Type  string `json:"type"`
	Data  any    `json:"data,omitempty"`
	Error string `json:"error,omitempty"`
}

type Hub struct {
	mu       sync.RWMutex
	rooms    map[int64]map[*Client]struct{}
	presence map[int64]map[int64]int
}

func NewHub() *Hub {
	return &Hub{
		rooms:    make(map[int64]map[*Client]struct{}),
		presence: make(map[int64]map[int64]int),
	}
}

func (h *Hub) Register(client *Client) bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, exists := h.rooms[client.ConversationID]; !exists {
		h.rooms[client.ConversationID] = make(map[*Client]struct{})
	}

	if _, exists := h.presence[client.ConversationID]; !exists {
		h.presence[client.ConversationID] = make(map[int64]int)
	}

	h.rooms[client.ConversationID][client] = struct{}{}
	h.presence[client.ConversationID][client.UserID]++

	return h.presence[client.ConversationID][client.UserID] == 1
}

func (h *Hub) Unregister(client *Client) bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	room, exists := h.rooms[client.ConversationID]
	if !exists {
		return false
	}

	if _, exists := room[client]; !exists {
		return false
	}

	delete(room, client)
	close(client.Send)

	lastConnection := false

	if users, exists := h.presence[client.ConversationID]; exists {
		users[client.UserID]--

		if users[client.UserID] <= 0 {
			delete(users, client.UserID)
			lastConnection = true
		}

		if len(users) == 0 {
			delete(h.presence, client.ConversationID)
		}
	}

	if len(room) == 0 {
		delete(h.rooms, client.ConversationID)
	}

	return lastConnection
}

func (h *Hub) Broadcast(conversationID int64, event Event) {
	h.broadcast(conversationID, nil, event)
}

func (h *Hub) BroadcastExcept(conversationID int64, except *Client, event Event) {
	h.broadcast(conversationID, except, event)
}

func (h *Hub) broadcast(conversationID int64, except *Client, event Event) {
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
		if client == except {
			continue
		}

		select {
		case client.Send <- payload:
		default:
		}
	}
}

func (h *Hub) OnlineUserIDs(conversationID int64) []int64 {
	h.mu.RLock()
	defer h.mu.RUnlock()

	users := h.presence[conversationID]
	result := make([]int64, 0, len(users))

	for userID := range users {
		result = append(result, userID)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i] < result[j]
	})

	return result
}
