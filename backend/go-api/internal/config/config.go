package config

import (
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Port            string
	DatabaseURL     string
	JWTSecret       string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
	LogLevel        string
	LogFormat       string
	// Shared with the frontend, which sends it with the visitor's IP (see middleware.ClientIP).
	ProxySecret string
	// Set when a proxy (Render) sits in front and appends the caller to X-Forwarded-For.
	BehindProxy bool
	// Per user, to protect the Earth Engine quota; 0 turns a limit off.
	SnapshotsPerDay   int
	MaxSnapshotAreaHa float64
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
		JWTSecret:       strings.TrimSpace(os.Getenv("JWT_SECRET")),
		AccessTokenTTL:  getDuration("ACCESS_TOKEN_TTL", 15*time.Minute),
		RefreshTokenTTL: getDuration("REFRESH_TOKEN_TTL", 30*24*time.Hour),
		LogLevel:        getEnv("LOG_LEVEL", "info"),
		LogFormat:       getEnv("LOG_FORMAT", "text"),
		ProxySecret:     getEnv("PROXY_SECRET", ""),
		BehindProxy:     getEnv("BEHIND_PROXY", "") == "true",

		SnapshotsPerDay:   int(getNumber("SNAPSHOTS_PER_DAY", 30)),
		MaxSnapshotAreaHa: getNumber("MAX_SNAPSHOT_AREA_HA", 50_000),
	}

	if cfg.DatabaseURL == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	// A short secret makes JWTs easy to brute-force.
	if len(cfg.JWTSecret) < 32 {
		log.Fatal("JWT_SECRET environment variable is required and must be at least 32 characters")
	}

	if cfg.ProxySecret != "" && len(cfg.ProxySecret) < 32 {
		log.Fatal("PROXY_SECRET must be at least 32 characters")
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

func getNumber(key string, fallback float64) float64 {
	value := getEnv(key, "")
	if value == "" {
		return fallback
	}
	n, err := strconv.ParseFloat(value, 64)
	if err != nil || n < 0 {
		log.Fatalf("%s must be a number of 0 or more: %q", key, value)
	}
	return n
}

func getEnv(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	return value
}
