package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/renaldid/chat-go/internal/service"
)

type MessageHandler struct {
	messageService *service.MessageService
}

func NewMessageHandler(
	messageService *service.MessageService,
) *MessageHandler {
	return &MessageHandler{
		messageService: messageService,
	}
}

type createMessageRequest struct {
	Content string `json:"content"`
}

func (h *MessageHandler) Create(c *gin.Context) {
	conversationID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || conversationID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid conversation id",
		})
		return
	}

	userID, ok := authenticatedUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "unauthorized",
		})
		return
	}

	var req createMessageRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	message, err := h.messageService.Create(
		c.Request.Context(),
		conversationID,
		userID,
		req.Content,
	)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrMessageContentRequired):
			c.JSON(http.StatusBadRequest, gin.H{
				"error": service.ErrMessageContentRequired.Error(),
			})

		case errors.Is(err, service.ErrConversationNotFound):
			c.JSON(http.StatusNotFound, gin.H{
				"error": service.ErrConversationNotFound.Error(),
			})

		case errors.Is(err, service.ErrUserNotFound):
			c.JSON(http.StatusNotFound, gin.H{
				"error": service.ErrUserNotFound.Error(),
			})

		case errors.Is(err, service.ErrSenderNotConversationMember):
			c.JSON(http.StatusForbidden, gin.H{
				"error": service.ErrSenderNotConversationMember.Error(),
			})

		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "internal server error",
			})
		}

		return
	}

	c.JSON(http.StatusCreated, message)
}

func (h *MessageHandler) GetByConversationID(c *gin.Context) {
	conversationID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || conversationID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid conversation id",
		})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	messages, err := h.messageService.GetByConversationID(
		c.Request.Context(),
		conversationID,
		limit,
		offset,
	)
	if err != nil {
		if errors.Is(err, service.ErrConversationNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": service.ErrConversationNotFound.Error(),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "internal server error",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": messages,
	})
}
