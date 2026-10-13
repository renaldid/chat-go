package websocket

import (
	"context"
	"log/slog"
	"time"
)

const presenceSyncPeriod = 30 * time.Second

func StartPresenceSync(
	ctx context.Context,
	hub *Hub,
	presence *RedisPresence,
) {
	ticker := time.NewTicker(presenceSyncPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			for _, conversationID := range hub.ConversationIDs() {
				if err := presence.Refresh(
					ctx,
					conversationID,
				); err != nil {
					slog.Error(
						"refresh redis presence",
						"conversation_id", conversationID,
						"error", err,
					)
					continue
				}

				users, err := presence.OnlineUserIDs(
					ctx,
					conversationID,
				)
				if err != nil {
					slog.Error(
						"get redis presence snapshot",
						"conversation_id", conversationID,
						"error", err,
					)
					continue
				}

				hub.Broadcast(
					conversationID,
					Event{
						Type: "presence_snapshot",
						Data: map[string]any{
							"conversation_id": conversationID,
							"online_user_ids": users,
						},
					},
				)
			}
		}
	}
}
