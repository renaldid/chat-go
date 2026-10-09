package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/renaldid/chat-go/internal/repository"
	"github.com/renaldid/chat-go/internal/service"
)

func NewRouter(db *pgxpool.Pool) *gin.Engine {
	r := gin.New()

	r.Use(gin.Recovery())
	r.Use(RequestLogger())

	healthHandler := NewHealthHandler(db)

	userRepository := repository.NewUserRepositoryPostgres(db)
	userService := service.NewUserService(userRepository)
	userHandler := NewUserHandler(userService)

	r.GET("/health", healthHandler.Check)

	users := r.Group("/users")
	{
		users.POST("", userHandler.Create)
		users.GET("/:id", userHandler.GetByID)
	}

	return r
}
