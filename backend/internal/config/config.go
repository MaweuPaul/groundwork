package config

import (
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Port        string
	DatabaseURL string
}

func Load() *Config {
	// Load local environment variables when a .env file exists.
	// In production, variables should normally come from the host.
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found; using system environment variables")
	}

	cfg := &Config{
		Port:        getEnv("PORT", "8080"),
		DatabaseURL: strings.TrimSpace(os.Getenv("DATABASE_URL")),
	}

	if cfg.DatabaseURL == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	return cfg
}

func getEnv(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	return value
}
