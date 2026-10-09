package handler

import "github.com/gin-gonic/gin"

func NewRouter() *gin.Engine {
	r := gin.New()

	r.Use(gin.Recovery())
	r.Use(RequestLogger())

	r.GET("/health", HealthCheck)

	return r
}
