package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func NewRouter(db *pgxpool.Pool) *gin.Engine {
	r := gin.New()

	r.Use(gin.Recovery())
	r.Use(RequestLogger())

	healthHandler := NewHealthHandler(db)

	r.GET("/health", healthHandler.Check)

	return r
}
