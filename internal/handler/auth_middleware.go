package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/renaldid/chat-go/internal/auth"
)

const authenticatedUserIDKey = "authenticated_user_id"

func AuthMiddleware(jwtManager *auth.JWTManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		authorization := strings.TrimSpace(
			c.GetHeader("Authorization"),
		)

		if authorization == "" {
			c.AbortWithStatusJSON(
				http.StatusUnauthorized,
				gin.H{
					"error": "authorization header is required",
				},
			)
			return
		}

		const bearerPrefix = "Bearer "

		if !strings.HasPrefix(authorization, bearerPrefix) {
			c.AbortWithStatusJSON(
				http.StatusUnauthorized,
				gin.H{
					"error": "invalid authorization header",
				},
			)
			return
		}

		tokenString := strings.TrimSpace(
			strings.TrimPrefix(authorization, bearerPrefix),
		)

		if tokenString == "" {
			c.AbortWithStatusJSON(
				http.StatusUnauthorized,
				gin.H{
					"error": "invalid access token",
				},
			)
			return
		}

		claims, err := jwtManager.Parse(tokenString)
		if err != nil {
			c.AbortWithStatusJSON(
				http.StatusUnauthorized,
				gin.H{
					"error": "invalid access token",
				},
			)
			return
		}

		c.Set(authenticatedUserIDKey, claims.UserID)

		c.Next()
	}
}

func authenticatedUserID(c *gin.Context) (int64, bool) {
	value, exists := c.Get(authenticatedUserIDKey)
	if !exists {
		return 0, false
	}

	userID, ok := value.(int64)
	if !ok || userID <= 0 {
		return 0, false
	}

	return userID, true
}
