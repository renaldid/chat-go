package websocket

import (
	"encoding/json"
	"testing"
)

func TestHubPresence(t *testing.T) {
	hub := NewHub()

	client1 := &Client{
		ConversationID: 1,
		UserID:         10,
		Send:           make(chan []byte, 10),
	}

	first := hub.Register(client1)
	if !first {
		t.Fatal("expected first connection to mark user online")
	}

	users := hub.OnlineUserIDs(1)
	if len(users) != 1 || users[0] != 10 {
		t.Fatalf("expected online user 10, got %v", users)
	}

	client2 := &Client{
		ConversationID: 1,
		UserID:         10,
		Send:           make(chan []byte, 10),
	}

	first = hub.Register(client2)
	if first {
		t.Fatal("expected second connection for same user not to mark new online state")
	}

	last := hub.Unregister(client1)
	if last {
		t.Fatal("expected user to remain online while second connection exists")
	}

	last = hub.Unregister(client2)
	if !last {
		t.Fatal("expected last connection to mark user offline")
	}

	users = hub.OnlineUserIDs(1)
	if len(users) != 0 {
		t.Fatalf("expected no online users, got %v", users)
	}
}

func TestHubBroadcast(t *testing.T) {
	hub := NewHub()

	client1 := &Client{
		ConversationID: 1,
		UserID:         1,
		Send:           make(chan []byte, 10),
	}

	client2 := &Client{
		ConversationID: 1,
		UserID:         2,
		Send:           make(chan []byte, 10),
	}

	hub.Register(client1)
	hub.Register(client2)

	hub.Broadcast(
		1,
		Event{
			Type: "test",
			Data: map[string]any{
				"value": "hello",
			},
		},
	)

	for _, client := range []*Client{client1, client2} {
		payload := <-client.Send

		var event Event
		if err := json.Unmarshal(payload, &event); err != nil {
			t.Fatalf("unmarshal event: %v", err)
		}

		if event.Type != "test" {
			t.Fatalf("expected test event, got %s", event.Type)
		}
	}

	hub.Unregister(client1)
	hub.Unregister(client2)
}

func TestHubBroadcastExcept(t *testing.T) {
	hub := NewHub()

	client1 := &Client{
		ConversationID: 1,
		UserID:         1,
		Send:           make(chan []byte, 10),
	}

	client2 := &Client{
		ConversationID: 1,
		UserID:         2,
		Send:           make(chan []byte, 10),
	}

	hub.Register(client1)
	hub.Register(client2)

	hub.BroadcastExcept(
		1,
		client1,
		Event{
			Type: "typing_start",
		},
	)

	select {
	case <-client1.Send:
		t.Fatal("expected excluded client not to receive event")
	default:
	}

	select {
	case <-client2.Send:
	default:
		t.Fatal("expected second client to receive event")
	}

	hub.Unregister(client1)
	hub.Unregister(client2)
}
