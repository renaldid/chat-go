package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/renaldid/chat-go/internal/service"
)

type ConversationHandler struct {
	conversationService *service.ConversationService
}

func NewConversationHandler(
	conversationService *service.ConversationService,
) *ConversationHandler {
	return &ConversationHandler{
		conversationService: conversationService,
	}
}

type createConversationRequest struct {
	Title string `json:"title"`
}

func (h *ConversationHandler) Create(c *gin.Context) {
	var req createConversationRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	conversation, err := h.conversationService.Create(
		c.Request.Context(),
		req.Title,
	)
	if err != nil {
		if errors.Is(err, service.ErrConversationTitleRequired) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": service.ErrConversationTitleRequired.Error(),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "internal server error",
		})
		return
	}

	c.JSON(http.StatusCreated, conversation)
}

func (h *ConversationHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid conversation id",
		})
		return
	}

	conversation, err := h.conversationService.GetByID(
		c.Request.Context(),
		id,
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

	c.JSON(http.StatusOK, conversation)
}
