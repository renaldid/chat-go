package main

import (
	"github.com/renaldid/chat-go/internal/config"
	"github.com/renaldid/chat-go/internal/handler"
)

func main() {
	cfg := config.Load()

	r := handler.NewRouter()

	if err := r.Run(":" + cfg.Port); err != nil {
		panic(err)
	}
}
