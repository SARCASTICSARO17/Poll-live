package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

// Config holds all application configuration loaded from environment variables.
type Config struct {
	Port          string
	MongoURI      string
	MongoDatabase string
	RedisURL      string
	JWTSecret     string
	FrontendURL   string
}

// Load reads configuration from the environment. It attempts to load a .env
// file if present, but real environment variables always take precedence.
func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		Port:          getEnv("PORT", "8080"),
		MongoURI:      os.Getenv("MONGO_URI"),
		MongoDatabase: getEnv("MONGO_DATABASE", "poll_live"),
		RedisURL:      os.Getenv("REDIS_URL"),
		JWTSecret:     os.Getenv("JWT_SECRET"),
		FrontendURL:   getEnv("FRONTEND_URL", "http://localhost:5173"),
	}

	if cfg.MongoURI == "" {
		return nil, fmt.Errorf("MONGO_URI is required")
	}
	if cfg.RedisURL == "" {
		return nil, fmt.Errorf("REDIS_URL is required")
	}
	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}