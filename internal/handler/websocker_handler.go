package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	gorilla "github.com/gorilla/websocket"

	"github.com/renaldid/chat-go/internal/auth"
	"github.com/renaldid/chat-go/internal/service"
	ws "github.com/renaldid/chat-go/internal/websocket"
)

type WebSocketHandler struct {
	hub                 *ws.Hub
	broker              *ws.RedisBroker
	presence            *ws.RedisPresence
	messageService      *service.MessageService
	conversationService *service.ConversationService
	jwtManager          *auth.JWTManager
	upgrader            gorilla.Upgrader
}

type websocketRequest struct {
	Type    string `json:"type"`
	Content string `json:"content,omitempty"`
}

func NewWebSocketHandler(
	hub *ws.Hub,
	broker *ws.RedisBroker,
	presence *ws.RedisPresence,
	messageService *service.MessageService,
	conversationService *service.ConversationService,
	jwtManager *auth.JWTManager,
	allowedOrigin string,
) *WebSocketHandler {
	return &WebSocketHandler{
		hub:                 hub,
		broker:              broker,
		presence:            presence,
		messageService:      messageService,
		conversationService: conversationService,
		jwtManager:          jwtManager,
		upgrader: gorilla.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				origin := r.Header.Get("Origin")

				return origin == "" ||
					allowedOrigin == "*" ||
					origin == allowedOrigin
			},
		},
	}
}

func (h *WebSocketHandler) Connect(c *gin.Context) {
	conversationID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || conversationID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid conversation id",
		})
		return
	}

	tokenString := c.Query("token")
	if tokenString == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "access token is required",
		})
		return
	}

	claims, err := h.jwtManager.Parse(tokenString)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "invalid access token",
		})
		return
	}

	_, err = h.conversationService.GetByID(
		c.Request.Context(),
		conversationID,
		claims.UserID,
	)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrConversationNotFound):
			c.JSON(
				http.StatusNotFound,
				gin.H{
					"error": service.ErrConversationNotFound.Error(),
				},
			)

		case errors.Is(err, service.ErrConversationAccessDenied):
			c.JSON(
				http.StatusForbidden,
				gin.H{
					"error": service.ErrConversationAccessDenied.Error(),
				},
			)

		default:
			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"error": "internal server error",
				},
			)
		}

		return
	}

	conn, err := h.upgrader.Upgrade(
		c.Writer,
		c.Request,
		nil,
	)
	if err != nil {
		return
	}

	client := ws.NewClient(
		conn,
		conversationID,
		claims.UserID,
	)

	client.ConfigureReadDeadline()

	h.hub.Register(client)

	becameOnline, err := h.presence.Register(
		c.Request.Context(),
		conversationID,
		claims.UserID,
	)
	if err != nil {
		h.hub.Unregister(client)
		_ = conn.Close()

		slog.Error(
			"register redis presence",
			"conversation_id", conversationID,
			"user_id", claims.UserID,
			"error", err,
		)

		return
	}

	defer func() {
		h.hub.Unregister(client)

		cleanupCtx, cancel := context.WithTimeout(
			context.Background(),
			3*time.Second,
		)
		defer cancel()

		becameOffline, err := h.presence.Unregister(
			cleanupCtx,
			conversationID,
			claims.UserID,
		)
		if err != nil {
			slog.Error(
				"unregister redis presence",
				"conversation_id", conversationID,
				"user_id", claims.UserID,
				"error", err,
			)
			return
		}

		if !becameOffline {
			return
		}

		event := ws.Event{
			Type: "user_offline",
			Data: map[string]any{
				"conversation_id": conversationID,
				"user_id":         claims.UserID,
			},
		}

		h.hub.Broadcast(
			conversationID,
			event,
		)

		if err := h.broker.Publish(
			cleanupCtx,
			conversationID,
			event,
		); err != nil {
			slog.Error(
				"publish user offline",
				"conversation_id", conversationID,
				"user_id", claims.UserID,
				"error", err,
			)
		}
	}()

	go client.WritePump()

	onlineUserIDs, err := h.presence.OnlineUserIDs(
		c.Request.Context(),
		conversationID,
	)
	if err != nil {
		slog.Error(
			"get redis online users",
			"conversation_id", conversationID,
			"error", err,
		)

		onlineUserIDs = h.hub.OnlineUserIDs(
			conversationID,
		)
	}

	client.SendEvent(ws.Event{
		Type: "connected",
		Data: map[string]any{
			"conversation_id": conversationID,
			"user_id":         claims.UserID,
			"online_user_ids": onlineUserIDs,
		},
	})

	if becameOnline {
		event := ws.Event{
			Type: "user_online",
			Data: map[string]any{
				"conversation_id": conversationID,
				"user_id":         claims.UserID,
			},
		}

		h.hub.BroadcastExcept(
			conversationID,
			client,
			event,
		)

		if err := h.broker.Publish(
			c.Request.Context(),
			conversationID,
			event,
		); err != nil {
			slog.Error(
				"publish user online",
				"conversation_id", conversationID,
				"user_id", claims.UserID,
				"error", err,
			)
		}
	}

	for {
		var req websocketRequest

		if err := conn.ReadJSON(&req); err != nil {
			return
		}

		switch req.Type {
		case "", "message":
			h.handleMessage(
				c,
				client,
				claims.UserID,
				conversationID,
				req.Content,
			)

		case "typing_start":
			h.handleTyping(
				client,
				claims.UserID,
				conversationID,
				true,
			)

		case "typing_stop":
			h.handleTyping(
				client,
				claims.UserID,
				conversationID,
				false,
			)

		default:
			client.SendEvent(ws.Event{
				Type:  "error",
				Error: "unsupported event type",
			})
		}
	}
}

func (h *WebSocketHandler) handleMessage(
	c *gin.Context,
	client *ws.Client,
	userID int64,
	conversationID int64,
	content string,
) {
	message, err := h.messageService.Create(
		c.Request.Context(),
		conversationID,
		userID,
		content,
	)
	if err != nil {
		client.SendEvent(ws.Event{
			Type:  "error",
			Error: err.Error(),
		})
		return
	}

	event := ws.Event{
		Type: "message",
		Data: message,
	}

	h.hub.Broadcast(
		conversationID,
		event,
	)

	if err := h.broker.Publish(
		c.Request.Context(),
		conversationID,
		event,
	); err != nil {
		slog.Error(
			"publish websocket message",
			"conversation_id", conversationID,
			"user_id", userID,
			"error", err,
		)
	}
}

func (h *WebSocketHandler) handleTyping(
	client *ws.Client,
	userID int64,
	conversationID int64,
	isTyping bool,
) {
	eventType := "typing_stop"

	if isTyping {
		eventType = "typing_start"
	}

	event := ws.Event{
		Type: eventType,
		Data: map[string]any{
			"conversation_id": conversationID,
			"user_id":         userID,
		},
	}

	h.hub.BroadcastExcept(
		conversationID,
		client,
		event,
	)

	publishCtx, cancel := context.WithTimeout(
		context.Background(),
		2*time.Second,
	)
	defer cancel()

	if err := h.broker.Publish(
		publishCtx,
		conversationID,
		event,
	); err != nil {
		slog.Error(
			"publish typing event",
			"conversation_id", conversationID,
			"user_id", userID,
			"error", err,
		)
	}
}
