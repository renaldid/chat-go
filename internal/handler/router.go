package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/renaldid/chat-go/internal/auth"
	"github.com/renaldid/chat-go/internal/config"
	"github.com/renaldid/chat-go/internal/repository"
	"github.com/renaldid/chat-go/internal/service"
	ws "github.com/renaldid/chat-go/internal/websocket"
)

func NewRouter(db *pgxpool.Pool, cfg config.Config) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(RequestLogger())

	healthHandler := NewHealthHandler(db)

	userRepository := repository.NewUserRepositoryPostgres(db)
	userService := service.NewUserService(userRepository)
	userHandler := NewUserHandler(userService)

	jwtManager := auth.NewJWTManager(cfg.JWTSecret, cfg.JWTExpiration)
	authService := service.NewAuthService(userRepository, jwtManager)
	authHandler := NewAuthHandler(authService)

	conversationRepository := repository.NewConversationRepositoryPostgres(db)
	conversationMemberRepository := repository.NewConversationMemberRepositoryPostgres(db)

	conversationService := service.NewConversationService(
		conversationRepository,
		conversationMemberRepository,
	)
	conversationHandler := NewConversationHandler(conversationService)

	conversationMemberService := service.NewConversationMemberService(
		conversationRepository,
		userRepository,
		conversationMemberRepository,
	)
	conversationMemberHandler := NewConversationMemberHandler(conversationMemberService)

	messageRepository := repository.NewMessageRepositoryPostgres(db)
	messageService := service.NewMessageService(
		messageRepository,
		conversationRepository,
		userRepository,
		conversationMemberRepository,
	)
	messageHandler := NewMessageHandler(messageService)

	readReceiptService := service.NewReadReceiptService(
		conversationRepository,
		conversationMemberRepository,
		messageRepository,
	)
	readReceiptHandler := NewReadReceiptHandler(readReceiptService)

	websocketHub := ws.NewHub()
	websocketHandler := NewWebSocketHandler(
		websocketHub,
		messageService,
		conversationService,
		jwtManager,
		cfg.WSAllowedOrigin,
	)

	r.GET("/health", healthHandler.Check)

	authRoutes := r.Group("/auth")
	{
		authRoutes.POST("/login", authHandler.Login)
	}

	users := r.Group("/users")
	{
		users.POST("", userHandler.Create)
		users.GET("/:id", userHandler.GetByID)
	}

	conversations := r.Group("/conversations")
	conversations.Use(AuthMiddleware(jwtManager))
	{
		conversations.POST("", conversationHandler.Create)
		conversations.GET("", conversationHandler.List)
		conversations.GET("/:id", conversationHandler.GetByID)
		conversations.POST("/:id/members", conversationMemberHandler.Add)
		conversations.POST("/:id/messages", messageHandler.Create)
		conversations.GET("/:id/messages", messageHandler.GetByConversationID)
		conversations.PATCH("/:id/read", readReceiptHandler.MarkRead)
	}

	r.GET("/ws/conversations/:id", websocketHandler.Connect)

	return r
}
