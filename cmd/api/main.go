package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/renaldid/chat-go/internal/config"
	"github.com/renaldid/chat-go/internal/database"
	"github.com/renaldid/chat-go/internal/handler"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error(
			"failed to load config",
			"error", err,
		)
		os.Exit(1)
	}

	appCtx, cancelApp := context.WithCancel(
		context.Background(),
	)
	defer cancelApp()

	db, err := database.NewPostgresPool(
		appCtx,
		cfg,
	)
	if err != nil {
		slog.Error(
			"failed to connect to database",
			"error", err,
		)
		os.Exit(1)
	}
	defer db.Close()

	slog.Info("database connected")

	redisClient, err := database.NewRedisClient(
		appCtx,
		cfg,
	)
	if err != nil {
		slog.Error(
			"failed to connect to redis",
			"error", err,
		)
		os.Exit(1)
	}

	defer func() {
		if err := redisClient.Close(); err != nil {
			slog.Error(
				"failed to close redis",
				"error", err,
			)
		}
	}()

	slog.Info("redis connected")

	router := handler.NewRouter(
		appCtx,
		db,
		redisClient,
		cfg,
	)

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		slog.Info(
			"server started",
			"port", cfg.Port,
			"instance_id", cfg.InstanceID,
		)

		if err := server.ListenAndServe(); err != nil &&
			!errors.Is(err, http.ErrServerClosed) {
			slog.Error(
				"server error",
				"error", err,
			)
		}
	}()

	quit := make(chan os.Signal, 1)

	signal.Notify(
		quit,
		syscall.SIGINT,
		syscall.SIGTERM,
	)

	<-quit

	slog.Info("shutting down server")

	cancelApp()

	shutdownCtx, cancelShutdown :=
		context.WithTimeout(
			context.Background(),
			10*time.Second,
		)

	defer cancelShutdown()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error(
			"server shutdown failed",
			"error", err,
		)
		return
	}

	slog.Info("server stopped")
}
