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
	hub                       *ws.Hub
	messageService            *service.MessageService
	conversationService       *service.ConversationService
	conversationMemberService *service.ConversationMemberService
	jwtManager                *auth.JWTManager
	upgrader                  gorilla.Upgrader
}

func NewWebSocketHandler(
	hub *ws.Hub,
	messageService *service.MessageService,
	conversationService *service.ConversationService,
	conversationMemberService *service.ConversationMemberService,
	jwtManager *auth.JWTManager,
	allowedOrigin string,
) *WebSocketHandler {
	return &WebSocketHandler{
		hub:                       hub,
		messageService:            messageService,
		conversationService:       conversationService,
		conversationMemberService: conversationMemberService,
		jwtManager:                jwtManager,
		upgrader: gorilla.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				origin := r.Header.Get("Origin")
				return origin == "" || allowedOrigin == "*" || origin == allowedOrigin
			},
		},
	}
}

type websocketMessageRequest struct {
	Content string `json:"content"`
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

	_, err = h.conversationService.GetByID(c.Request.Context(), conversationID)
	if err != nil {
		if errors.Is(err, service.ErrConversationNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": service.ErrConversationNotFound.Error()})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	isMember, err := h.conversationMemberService.IsMember(c.Request.Context(), conversationID, claims.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	if !isMember {
		c.JSON(http.StatusForbidden, gin.H{"error": "user is not a conversation member"})
		return
	}

	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	client := ws.NewClient(conn, conversationID, claims.UserID)
	h.hub.Register(client)
	defer h.hub.Unregister(client)

	go client.WritePump()

	client.SendEvent(ws.Event{
		Type: "connected",
		Data: map[string]any{
			"conversation_id": conversationID,
			"user_id":         claims.UserID,
		},
	})

	for {
		var req websocketMessageRequest

		if err := conn.ReadJSON(&req); err != nil {
			return
		}

		message, err := h.messageService.Create(
			c.Request.Context(),
			conversationID,
			claims.UserID,
			req.Content,
		)
		if err != nil {
			client.SendEvent(ws.Event{
				Type:  "error",
				Error: err.Error(),
			})
			continue
		}

		h.hub.Broadcast(conversationID, ws.Event{
			Type: "message",
			Data: message,
		})
	}
}
