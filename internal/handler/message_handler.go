package handler

import (
	"errors"
	ws "github.com/renaldid/chat-go/internal/websocket"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/renaldid/chat-go/internal/service"
)

type MessageHandler struct {
	messageService *service.MessageService
	hub            *ws.Hub
	broker         *ws.RedisBroker
}

func NewMessageHandler(messageService *service.MessageService, hub *ws.Hub, broker *ws.RedisBroker) *MessageHandler {
	return &MessageHandler{
		messageService: messageService,
		hub:            hub,
		broker:         broker,
	}
}

type createMessageRequest struct {
	Content          string `json:"content"`
	ReplyToMessageID *int64 `json:"reply_to_message_id,omitempty"`
}
type updateMessageRequest struct {
	Content string `json:"content"`
}

func (h *MessageHandler) Create(c *gin.Context) {
	userID, ok := authenticatedUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	conversationID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || conversationID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid conversation id"})
		return
	}

	var req createMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		slog.Warn("create message invalid request",
			"conversation_id", conversationID,
			"user_id", userID,
			"error", err,
		)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	message, err := h.messageService.CreateWithReply(
		c.Request.Context(),
		conversationID,
		userID,
		req.Content,
		req.ReplyToMessageID,
	)
	if err != nil {
		writeCreateMessageError(c, conversationID, userID, req.ReplyToMessageID, err)
		return
	}

	event := ws.Event{
		Type: "message",
		Data: message,
	}

	h.hub.Broadcast(conversationID, event)

	if err := h.broker.Publish(c.Request.Context(), conversationID, event); err != nil {
		slog.Error("publish rest message failed",
			"conversation_id", conversationID,
			"message_id", message.ID,
			"user_id", userID,
			"error", err,
		)
	}

	slog.Info("message created",
		"conversation_id", conversationID,
		"message_id", message.ID,
		"user_id", userID,
		"reply_to_message_id", req.ReplyToMessageID,
	)

	c.JSON(http.StatusCreated, message)
}

func (h *MessageHandler) Update(c *gin.Context) {
	userID, ok := authenticatedUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	conversationID, messageID, ok := messageMutationIDs(c)
	if !ok {
		return
	}

	var req updateMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	message, err := h.messageService.Update(c.Request.Context(), conversationID, userID, messageID, req.Content)
	if err != nil {
		writeMessageMutationError(c, "update", conversationID, messageID, userID, err)
		return
	}

	event := ws.Event{Type: "message_updated", Data: message}
	h.hub.Broadcast(conversationID, event)

	if err := h.broker.Publish(c.Request.Context(), conversationID, event); err != nil {
		slog.Error("publish message updated failed",
			"conversation_id", conversationID,
			"message_id", messageID,
			"user_id", userID,
			"error", err,
		)
	}

	c.JSON(http.StatusOK, message)
}

func (h *MessageHandler) Delete(c *gin.Context) {
	userID, ok := authenticatedUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	conversationID, messageID, ok := messageMutationIDs(c)
	if !ok {
		return
	}

	message, changed, err := h.messageService.Delete(c.Request.Context(), conversationID, userID, messageID)
	if err != nil {
		writeMessageMutationError(c, "delete", conversationID, messageID, userID, err)
		return
	}

	if changed {
		event := ws.Event{Type: "message_deleted", Data: message}
		h.hub.Broadcast(conversationID, event)

		if err := h.broker.Publish(c.Request.Context(), conversationID, event); err != nil {
			slog.Error("publish message deleted failed",
				"conversation_id", conversationID,
				"message_id", messageID,
				"user_id", userID,
				"error", err,
			)
		}
	}

	c.Status(http.StatusNoContent)
}

func (h *MessageHandler) GetByConversationID(c *gin.Context) {
	userID, ok := authenticatedUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	conversationID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || conversationID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid conversation id"})
		return
	}

	limit, err := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if err != nil || limit <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid limit"})
		return
	}

	var cursor *int64

	if rawCursor := c.Query("cursor"); rawCursor != "" {
		parsedCursor, err := strconv.ParseInt(rawCursor, 10, 64)
		if err != nil || parsedCursor <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid cursor"})
			return
		}

		cursor = &parsedCursor
	}

	page, err := h.messageService.GetByConversationID(
		c.Request.Context(),
		conversationID,
		userID,
		cursor,
		limit,
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

	c.JSON(http.StatusOK, gin.H{
		"data":        page.Messages,
		"next_cursor": page.NextCursor,
		"has_more":    page.HasMore,
	})
}

func messageMutationIDs(c *gin.Context) (int64, int64, bool) {
	conversationID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || conversationID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid conversation id"})
		return 0, 0, false
	}

	messageID, err := strconv.ParseInt(c.Param("message_id"), 10, 64)
	if err != nil || messageID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid message id"})
		return 0, 0, false
	}

	return conversationID, messageID, true
}

func writeMessageMutationError(c *gin.Context, action string, conversationID int64, messageID int64, userID int64, err error) {
	switch {
	case errors.Is(err, service.ErrConversationNotFound), errors.Is(err, service.ErrMessageNotFound):
		slog.Warn("message mutation rejected",
			"action", action,
			"conversation_id", conversationID,
			"message_id", messageID,
			"user_id", userID,
			"error", err,
		)
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})

	case errors.Is(err, service.ErrConversationAccessDenied), errors.Is(err, service.ErrMessagePermissionDenied):
		slog.Warn("message mutation rejected",
			"action", action,
			"conversation_id", conversationID,
			"message_id", messageID,
			"user_id", userID,
			"error", err,
		)
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})

	case errors.Is(err, service.ErrMessageContentRequired), errors.Is(err, service.ErrMessageNotInConversation):
		slog.Warn("message mutation rejected",
			"action", action,
			"conversation_id", conversationID,
			"message_id", messageID,
			"user_id", userID,
			"error", err,
		)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})

	case errors.Is(err, service.ErrMessageCannotEditDeleted):
		slog.Warn("message mutation rejected",
			"action", action,
			"conversation_id", conversationID,
			"message_id", messageID,
			"user_id", userID,
			"error", err,
		)
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})

	default:
		slog.Error("message mutation failed",
			"action", action,
			"conversation_id", conversationID,
			"message_id", messageID,
			"user_id", userID,
			"error", err,
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	}
}

func writeCreateMessageError(c *gin.Context, conversationID int64, userID int64, replyToMessageID *int64, err error) {
	switch {
	case errors.Is(err, service.ErrConversationNotFound),
		errors.Is(err, service.ErrUserNotFound),
		errors.Is(err, service.ErrReplyMessageNotFound):
		slog.Warn("create message rejected",
			"conversation_id", conversationID,
			"user_id", userID,
			"reply_to_message_id", replyToMessageID,
			"error", err,
		)
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})

	case errors.Is(err, service.ErrSenderNotConversationMember),
		errors.Is(err, service.ErrConversationAccessDenied):
		slog.Warn("create message rejected",
			"conversation_id", conversationID,
			"user_id", userID,
			"reply_to_message_id", replyToMessageID,
			"error", err,
		)
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})

	case errors.Is(err, service.ErrMessageContentRequired),
		errors.Is(err, service.ErrReplyMessageNotInConversation):
		slog.Warn("create message rejected",
			"conversation_id", conversationID,
			"user_id", userID,
			"reply_to_message_id", replyToMessageID,
			"error", err,
		)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})

	case errors.Is(err, service.ErrCannotReplyDeletedMessage):
		slog.Warn("create message rejected",
			"conversation_id", conversationID,
			"user_id", userID,
			"reply_to_message_id", replyToMessageID,
			"error", err,
		)
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})

	default:
		slog.Error("create message failed",
			"conversation_id", conversationID,
			"user_id", userID,
			"reply_to_message_id", replyToMessageID,
			"error", err,
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	}
}
