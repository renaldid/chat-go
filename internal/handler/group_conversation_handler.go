package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/renaldid/chat-go/internal/service"
)

type GroupConversationHandler struct {
	service *service.GroupConversationService
}

func NewGroupConversationHandler(
	service *service.GroupConversationService,
) *GroupConversationHandler {
	return &GroupConversationHandler{service: service}
}

type createGroupRequest struct {
	Title     string  `json:"title"`
	MemberIDs []int64 `json:"member_ids"`
}

type renameGroupRequest struct {
	Title string `json:"title"`
}

type addGroupMemberRequest struct {
	UserID int64 `json:"user_id"`
}

type updateGroupRoleRequest struct {
	Role string `json:"role"`
}

func (h *GroupConversationHandler) Create(c *gin.Context) {
	userID, ok := authenticatedUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req createGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	conversation, err := h.service.Create(
		c.Request.Context(),
		userID,
		req.Title,
		req.MemberIDs,
	)
	if err != nil {
		writeGroupError(c, err)
		return
	}

	c.JSON(http.StatusCreated, conversation)
}

func (h *GroupConversationHandler) Rename(c *gin.Context) {
	userID, ok := authenticatedUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	conversationID, ok := groupConversationID(c)
	if !ok {
		return
	}

	var req renameGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	conversation, err := h.service.Rename(
		c.Request.Context(),
		userID,
		conversationID,
		req.Title,
	)
	if err != nil {
		writeGroupError(c, err)
		return
	}

	c.JSON(http.StatusOK, conversation)
}

func (h *GroupConversationHandler) ListMembers(c *gin.Context) {
	userID, ok := authenticatedUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	conversationID, ok := groupConversationID(c)
	if !ok {
		return
	}

	members, err := h.service.ListMembers(
		c.Request.Context(),
		userID,
		conversationID,
	)
	if err != nil {
		writeGroupError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": members})
}

func (h *GroupConversationHandler) AddMember(c *gin.Context) {
	userID, ok := authenticatedUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	conversationID, ok := groupConversationID(c)
	if !ok {
		return
	}

	var req addGroupMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.UserID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	member, err := h.service.AddMember(
		c.Request.Context(),
		userID,
		conversationID,
		req.UserID,
	)
	if err != nil {
		writeGroupError(c, err)
		return
	}

	c.JSON(http.StatusCreated, member)
}

func (h *GroupConversationHandler) RemoveMember(c *gin.Context) {
	userID, ok := authenticatedUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	conversationID, ok := groupConversationID(c)
	if !ok {
		return
	}

	targetUserID, err := strconv.ParseInt(c.Param("user_id"), 10, 64)
	if err != nil || targetUserID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	if err := h.service.RemoveMember(
		c.Request.Context(),
		userID,
		conversationID,
		targetUserID,
	); err != nil {
		writeGroupError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *GroupConversationHandler) UpdateRole(c *gin.Context) {
	userID, ok := authenticatedUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	conversationID, ok := groupConversationID(c)
	if !ok {
		return
	}

	targetUserID, err := strconv.ParseInt(c.Param("user_id"), 10, 64)
	if err != nil || targetUserID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	var req updateGroupRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if err := h.service.UpdateRole(
		c.Request.Context(),
		userID,
		conversationID,
		targetUserID,
		req.Role,
	); err != nil {
		writeGroupError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *GroupConversationHandler) Leave(c *gin.Context) {
	userID, ok := authenticatedUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	conversationID, ok := groupConversationID(c)
	if !ok {
		return
	}

	if err := h.service.Leave(
		c.Request.Context(),
		userID,
		conversationID,
	); err != nil {
		writeGroupError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func groupConversationID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid conversation id"})
		return 0, false
	}

	return id, true
}

func writeGroupError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrConversationNotFound),
		errors.Is(err, service.ErrConversationMemberNotFound),
		errors.Is(err, service.ErrUserNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})

	case errors.Is(err, service.ErrConversationAccessDenied),
		errors.Is(err, service.ErrGroupPermissionDenied):
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})

	case errors.Is(err, service.ErrConversationMemberExists):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})

	case errors.Is(err, service.ErrConversationTitleRequired),
		errors.Is(err, service.ErrGroupConversationRequired),
		errors.Is(err, service.ErrCannotRemoveGroupOwner),
		errors.Is(err, service.ErrCannotChangeGroupOwnerRole),
		errors.Is(err, service.ErrInvalidConversationRole),
		errors.Is(err, service.ErrUseLeaveConversation):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})

	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	}
}
