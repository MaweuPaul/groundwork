package config

import (
	"log"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Port            string
	DatabaseURL     string
	GEEServiceURL   string
	JWTSecret       string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
}

func Load() *Config {
	// Load local environment variables when a .env file exists.
	// In production, variables should normally come from the host.
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found; using system environment variables")
	}

	cfg := &Config{
		Port:            getEnv("PORT", "8080"),
		DatabaseURL:     strings.TrimSpace(os.Getenv("DATABASE_URL")),
		GEEServiceURL:   getEnv("GEE_SERVICE_URL", "http://localhost:8000"),
		JWTSecret:       strings.TrimSpace(os.Getenv("JWT_SECRET")),
		AccessTokenTTL:  getDuration("ACCESS_TOKEN_TTL", 15*time.Minute),
		RefreshTokenTTL: getDuration("REFRESH_TOKEN_TTL", 30*24*time.Hour),
	}

	if cfg.DatabaseURL == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	// A short secret makes JWTs easy to brute-force.
	if len(cfg.JWTSecret) < 32 {
		log.Fatal("JWT_SECRET environment variable is required and must be at least 32 characters")
	}

	return cfg
}

// getDuration reads values like "15m" or "720h".
func getDuration(key string, fallback time.Duration) time.Duration {
	value := getEnv(key, "")
	if value == "" {
		return fallback
	}

	d, err := time.ParseDuration(value)
	if err != nil {
		log.Fatalf("%s must be a duration like 15m or 720h: %v", key, err)
	}

	return d
}

func getEnv(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	return value
}
