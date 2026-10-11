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

	conversationRepository := repository.NewConversationRepositoryPostgres(db)
	conversationService := service.NewConversationService(conversationRepository)
	conversationHandler := NewConversationHandler(conversationService)

	conversationMemberRepository := repository.NewConversationMemberRepositoryPostgres(db)
	conversationMemberService := service.NewConversationMemberService(
		conversationRepository,
		userRepository,
		conversationMemberRepository,
	)
	conversationMemberHandler := NewConversationMemberHandler(
		conversationMemberService,
	)

	messageRepository := repository.NewMessageRepositoryPostgres(db)
	messageService := service.NewMessageService(
		messageRepository,
		conversationRepository,
		userRepository,
		conversationMemberRepository,
	)
	messageHandler := NewMessageHandler(messageService)

	r.GET("/health", healthHandler.Check)

	users := r.Group("/users")
	{
		users.POST("", userHandler.Create)
		users.GET("/:id", userHandler.GetByID)
	}

	conversations := r.Group("/conversations")
	{
		conversations.POST("", conversationHandler.Create)
		conversations.GET("/:id", conversationHandler.GetByID)

		conversations.POST("/:id/members", conversationMemberHandler.Add)

		conversations.POST("/:id/messages", messageHandler.Create)
		conversations.GET("/:id/messages", messageHandler.GetByConversationID)
	}

	return r
}
