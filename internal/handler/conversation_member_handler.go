package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/renaldid/chat-go/internal/service"
)

type ConversationMemberHandler struct {
	conversationMemberService *service.ConversationMemberService
}

func NewConversationMemberHandler(
	conversationMemberService *service.ConversationMemberService,
) *ConversationMemberHandler {
	return &ConversationMemberHandler{
		conversationMemberService: conversationMemberService,
	}
}

type addConversationMemberRequest struct {
	UserID int64 `json:"user_id"`
}

func (h *ConversationMemberHandler) Add(c *gin.Context) {
	conversationID, err := strconv.ParseInt(
		c.Param("id"),
		10,
		64,
	)
	if err != nil || conversationID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid conversation id",
		})
		return
	}

	var req addConversationMemberRequest

	if err := c.ShouldBindJSON(&req); err != nil || req.UserID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	member, err := h.conversationMemberService.Add(
		c.Request.Context(),
		conversationID,
		req.UserID,
	)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrConversationNotFound):
			c.JSON(http.StatusNotFound, gin.H{
				"error": service.ErrConversationNotFound.Error(),
			})

		case errors.Is(err, service.ErrUserNotFound):
			c.JSON(http.StatusNotFound, gin.H{
				"error": service.ErrUserNotFound.Error(),
			})

		case errors.Is(err, service.ErrConversationMemberExists):
			c.JSON(http.StatusConflict, gin.H{
				"error": service.ErrConversationMemberExists.Error(),
			})

		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "internal server error",
			})
		}

		return
	}

	c.JSON(http.StatusCreated, member)
}
