package config

import (
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Port            string
	DBHost          string
	DBPort          string
	DBName          string
	DBUser          string
	DBPassword      string
	DBSSLMode       string
	JWTSecret       string
	JWTExpiration   time.Duration
	WSAllowedOrigin string
	RedisAddr       string
	RedisPassword   string
	RedisDB         int
	InstanceID      string
}

func Load() (Config, error) {
	_ = godotenv.Load()

	jwtExpiration, err := time.ParseDuration(getEnv("JWT_EXPIRATION", "24h"))
	if err != nil {
		jwtExpiration = 24 * time.Hour
	}

	redisDB, err := strconv.Atoi(getEnv("REDIS_DB", "0"))
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		Port:            getEnv("PORT", "8080"),
		DBHost:          getEnv("DB_HOST", "localhost"),
		DBPort:          getEnv("DB_PORT", "5432"),
		DBName:          getEnv("DB_NAME", "chat_go"),
		DBUser:          getEnv("DB_USER", "chat_go"),
		DBPassword:      getEnv("DB_PASSWORD", "chat_go"),
		DBSSLMode:       getEnv("DB_SSLMODE", "disable"),
		JWTSecret:       getEnv("JWT_SECRET", "change-me"),
		JWTExpiration:   jwtExpiration,
		WSAllowedOrigin: getEnv("WS_ALLOWED_ORIGIN", "http://localhost:3000"),
		RedisAddr:       getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:   getEnv("REDIS_PASSWORD", ""),
		RedisDB:         redisDB,
		InstanceID:      getEnv("INSTANCE_ID", "chat-go-local"),
	}
	return cfg, nil
}

func getEnv(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
