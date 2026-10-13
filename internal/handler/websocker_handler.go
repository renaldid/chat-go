package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	gorilla "github.com/gorilla/websocket"

	"github.com/renaldid/chat-go/internal/auth"
	"github.com/renaldid/chat-go/internal/service"
	ws "github.com/renaldid/chat-go/internal/websocket"
)

type WebSocketHandler struct {
	hub                 *ws.Hub
	messageService      *service.MessageService
	conversationService *service.ConversationService
	jwtManager          *auth.JWTManager
	upgrader            gorilla.Upgrader
}

func NewWebSocketHandler(
	hub *ws.Hub,
	messageService *service.MessageService,
	conversationService *service.ConversationService,
	jwtManager *auth.JWTManager,
	allowedOrigin string,
) *WebSocketHandler {
	return &WebSocketHandler{
		hub:                 hub,
		messageService:      messageService,
		conversationService: conversationService,
		jwtManager:          jwtManager,
		upgrader: gorilla.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				origin := r.Header.Get("Origin")
				return origin == "" || allowedOrigin == "*" || origin == allowedOrigin
			},
		},
	}
}

type websocketRequest struct {
	Type    string `json:"type"`
	Content string `json:"content,omitempty"`
}

func (h *WebSocketHandler) Connect(c *gin.Context) {
	conversationID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || conversationID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid conversation id"})
		return
	}

	tokenString := c.Query("token")
	if tokenString == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "access token is required"})
		return
	}

	claims, err := h.jwtManager.Parse(tokenString)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid access token"})
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
			c.JSON(http.StatusNotFound, gin.H{"error": service.ErrConversationNotFound.Error()})
		case errors.Is(err, service.ErrConversationAccessDenied):
			c.JSON(http.StatusForbidden, gin.H{"error": service.ErrConversationAccessDenied.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		}
		return
	}

	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	client := ws.NewClient(conn, conversationID, claims.UserID)
	client.ConfigureReadDeadline()

	firstConnection := h.hub.Register(client)

	defer func() {
		lastConnection := h.hub.Unregister(client)

		if lastConnection {
			h.hub.Broadcast(
				conversationID,
				ws.Event{
					Type: "user_offline",
					Data: map[string]any{
						"conversation_id": conversationID,
						"user_id":         claims.UserID,
					},
				},
			)
		}
	}()

	go client.WritePump()

	client.SendEvent(ws.Event{
		Type: "connected",
		Data: map[string]any{
			"conversation_id": conversationID,
			"user_id":         claims.UserID,
			"online_user_ids": h.hub.OnlineUserIDs(conversationID),
		},
	})

	if firstConnection {
		h.hub.BroadcastExcept(
			conversationID,
			client,
			ws.Event{
				Type: "user_online",
				Data: map[string]any{
					"conversation_id": conversationID,
					"user_id":         claims.UserID,
				},
			},
		)
	}

	for {
		var req websocketRequest

		if err := conn.ReadJSON(&req); err != nil {
			return
		}

		switch req.Type {
		case "", "message":
			h.handleMessage(c, client, claims.UserID, conversationID, req.Content)

		case "typing_start":
			h.handleTyping(client, claims.UserID, conversationID, true)

		case "typing_stop":
			h.handleTyping(client, claims.UserID, conversationID, false)

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

	h.hub.Broadcast(
		conversationID,
		ws.Event{
			Type: "message",
			Data: message,
		},
	)
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

	h.hub.BroadcastExcept(
		conversationID,
		client,
		ws.Event{
			Type: eventType,
			Data: map[string]any{
				"conversation_id": conversationID,
				"user_id":         userID,
			},
		},
	)
}
