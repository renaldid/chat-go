package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/renaldid/chat-go/internal/service"
)

type ReadReceiptHandler struct {
	readReceiptService *service.ReadReceiptService
}

func NewReadReceiptHandler(readReceiptService *service.ReadReceiptService) *ReadReceiptHandler {
	return &ReadReceiptHandler{readReceiptService: readReceiptService}
}

type markReadRequest struct {
	MessageID int64 `json:"message_id"`
}

func (h *ReadReceiptHandler) MarkRead(c *gin.Context) {
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

	var req markReadRequest

	if err := c.ShouldBindJSON(&req); err != nil || req.MessageID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	receipt, err := h.readReceiptService.MarkRead(
		c.Request.Context(),
		conversationID,
		userID,
		req.MessageID,
	)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrConversationNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": service.ErrConversationNotFound.Error()})
		case errors.Is(err, service.ErrConversationAccessDenied):
			c.JSON(http.StatusForbidden, gin.H{"error": service.ErrConversationAccessDenied.Error()})
		case errors.Is(err, service.ErrMessageNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": service.ErrMessageNotFound.Error()})
		case errors.Is(err, service.ErrMessageNotInConversation):
			c.JSON(http.StatusBadRequest, gin.H{"error": service.ErrMessageNotInConversation.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		}
		return
	}

	c.JSON(http.StatusOK, receipt)
}
