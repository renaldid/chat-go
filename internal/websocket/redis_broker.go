package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/redis/go-redis/v9"
)

const redisEventChannel = "chat:events"

type DistributedEvent struct {
	Origin         string `json:"origin"`
	ConversationID int64  `json:"conversation_id"`
	Event          Event  `json:"event"`
}

type RedisBroker struct {
	client     *redis.Client
	instanceID string
	hub        *Hub
}

func NewRedisBroker(
	client *redis.Client,
	instanceID string,
	hub *Hub,
) *RedisBroker {
	return &RedisBroker{
		client:     client,
		instanceID: instanceID,
		hub:        hub,
	}
}

func (b *RedisBroker) Start(ctx context.Context) error {
	pubsub := b.client.Subscribe(ctx, redisEventChannel)

	if _, err := pubsub.Receive(ctx); err != nil {
		_ = pubsub.Close()
		return fmt.Errorf("subscribe redis realtime events: %w", err)
	}

	go b.consume(ctx, pubsub)

	return nil
}

func (b *RedisBroker) consume(
	ctx context.Context,
	pubsub *redis.PubSub,
) {
	defer func() {
		_ = pubsub.Close()
	}()

	channel := pubsub.Channel()

	for {
		select {
		case <-ctx.Done():
			return

		case message, ok := <-channel:
			if !ok {
				return
			}

			var distributedEvent DistributedEvent

			if err := json.Unmarshal(
				[]byte(message.Payload),
				&distributedEvent,
			); err != nil {
				slog.Error(
					"decode redis realtime event",
					"error", err,
				)
				continue
			}

			if distributedEvent.Origin == b.instanceID {
				continue
			}

			b.hub.Broadcast(
				distributedEvent.ConversationID,
				distributedEvent.Event,
			)
		}
	}
}

func (b *RedisBroker) Publish(
	ctx context.Context,
	conversationID int64,
	event Event,
) error {
	payload, err := json.Marshal(DistributedEvent{
		Origin:         b.instanceID,
		ConversationID: conversationID,
		Event:          event,
	})
	if err != nil {
		return fmt.Errorf("encode redis realtime event: %w", err)
	}

	if err := b.client.Publish(
		ctx,
		redisEventChannel,
		payload,
	).Err(); err != nil {
		return fmt.Errorf("publish redis realtime event: %w", err)
	}

	return nil
}
