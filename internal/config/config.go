package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port string

	DBHost     string
	DBPort     string
	DBName     string
	DBUser     string
	DBPassword string
	DBSSLMode  string
}

func Load() Config {
	_ = godotenv.Load()

	port := getEnv("PORT", "8080")

	return Config{
		Port: port,

		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBName:     getEnv("DB_NAME", "chat_go"),
		DBUser:     getEnv("DB_USER", "chat_go"),
		DBPassword: getEnv("DB_PASSWORD", "chat_go"),
		DBSSLMode:  getEnv("DB_SSLMODE", "disable"),
	}
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	return value
}
