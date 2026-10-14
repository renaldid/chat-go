package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/renaldid/chat-go/internal/service"
)

type DirectConversationHandler struct {
	directConversationService *service.DirectConversationService
}

func NewDirectConversationHandler(
	directConversationService *service.DirectConversationService,
) *DirectConversationHandler {
	return &DirectConversationHandler{
		directConversationService: directConversationService,
	}
}

type createDirectConversationRequest struct {
	UserID int64 `json:"user_id"`
}

func (h *DirectConversationHandler) CreateOrGet(c *gin.Context) {
	userID, ok := authenticatedUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "unauthorized",
		})
		return
	}

	var req createDirectConversationRequest
	if err := c.ShouldBindJSON(&req); err != nil ||
		req.UserID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	conversation, created, err :=
		h.directConversationService.CreateOrGet(
			c.Request.Context(),
			userID,
			req.UserID,
		)

	if err != nil {
		switch {
		case errors.Is(err, service.ErrDirectConversationWithSelf):
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})

		case errors.Is(err, service.ErrUserNotFound):
			c.JSON(http.StatusNotFound, gin.H{
				"error": err.Error(),
			})

		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "internal server error",
			})
		}

		return
	}

	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}

	c.JSON(status, conversation)
}
