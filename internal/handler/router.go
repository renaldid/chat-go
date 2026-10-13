package handler

import (
	"context"
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/renaldid/chat-go/internal/auth"
	"github.com/renaldid/chat-go/internal/config"
	"github.com/renaldid/chat-go/internal/repository"
	"github.com/renaldid/chat-go/internal/service"
	ws "github.com/renaldid/chat-go/internal/websocket"
)

func NewRouter(ctx context.Context, db *pgxpool.Pool, redisClient *redis.Client, cfg config.Config) *gin.Engine {
	router := gin.New()

	router.Use(
		RequestLogger(),
		gin.Recovery(),
	)

	userRepository := repository.NewUserRepositoryPostgres(db)
	conversationRepository := repository.NewConversationRepositoryPostgres(db)
	conversationMemberRepository := repository.NewConversationMemberRepositoryPostgres(db)
	messageRepository := repository.NewMessageRepositoryPostgres(db)
	jwtManager := auth.NewJWTManager(cfg.JWTSecret, cfg.JWTExpiration)

	userService := service.NewUserService(userRepository)
	authService := service.NewAuthService(userRepository, jwtManager)
	conversationService := service.NewConversationService(conversationRepository, conversationMemberRepository)
	conversationMemberService := service.NewConversationMemberService(conversationRepository, userRepository, conversationMemberRepository)
	messageService := service.NewMessageService(messageRepository, conversationRepository, userRepository, conversationMemberRepository)
	readReceiptService := service.NewReadReceiptService(conversationRepository, conversationMemberRepository, messageRepository)

	websocketHub := ws.NewHub()

	redisBroker := ws.NewRedisBroker(redisClient, cfg.InstanceID, websocketHub)
	redisPresence := ws.NewRedisPresence(redisClient, cfg.InstanceID)
	if err := redisBroker.Start(ctx); err != nil {
		slog.Error("start redis realtime broker", "error", err)
	}

	go ws.StartPresenceSync(ctx, websocketHub, redisPresence)

	healthHandler := NewHealthHandler(db)
	userHandler := NewUserHandler(userService)
	authHandler := NewAuthHandler(authService)
	conversationHandler := NewConversationHandler(conversationService)
	conversationMemberHandler := NewConversationMemberHandler(conversationMemberService)
	messageHandler := NewMessageHandler(messageService)
	readReceiptHandler := NewReadReceiptHandler(readReceiptService)
	websocketHandler := NewWebSocketHandler(
		websocketHub,
		redisBroker,
		redisPresence,
		messageService,
		conversationService,
		jwtManager,
		cfg.WSAllowedOrigin,
	)

	router.GET("/health", healthHandler.Check)

	authGroup := router.Group("/auth")
	{
		authGroup.POST("/login", authHandler.Login)
	}

	users := router.Group("/users")
	{
		users.POST("", userHandler.Create)
		users.GET("/:id", userHandler.GetByID)
	}

	conversations := router.Group("/conversations")
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

	router.GET("/ws/conversations/:id", websocketHandler.Connect)

	return router
}
