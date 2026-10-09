package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/renaldid/chat-go/internal/service"
)

type UserHandler struct {
	userService *service.UserService
}

func NewUserHandler(userService *service.UserService) *UserHandler {
	return &UserHandler{
		userService: userService,
	}
}

type createUserRequest struct {
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
}

func (h *UserHandler) Create(c *gin.Context) {
	var req createUserRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	user, err := h.userService.Create(
		c.Request.Context(),
		req.Username,
		req.DisplayName,
	)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrUsernameRequired):
			c.JSON(http.StatusBadRequest, gin.H{
				"error": service.ErrUsernameRequired.Error(),
			})
		case errors.Is(err, service.ErrDisplayNameRequired):
			c.JSON(http.StatusBadRequest, gin.H{
				"error": service.ErrDisplayNameRequired.Error(),
			})
		case errors.Is(err, service.ErrUsernameExists):
			c.JSON(http.StatusConflict, gin.H{
				"error": service.ErrUsernameExists.Error(),
			})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "internal server error",
			})
		}

		return
	}

	c.JSON(http.StatusCreated, user)
}

func (h *UserHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid user id",
		})
		return
	}

	user, err := h.userService.GetByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": service.ErrUserNotFound.Error(),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "internal server error",
		})
		return
	}

	c.JSON(http.StatusOK, user)
}
