package main

import (
	"log"
	"log/slog"
	"net/http"
	"os"

	"groundwork/internal/app"
	"groundwork/internal/config"
	"groundwork/internal/database"
	"groundwork/internal/logging"
	"groundwork/internal/routes"
)

func main() {
	cfg := config.Load()

	if err := logging.Setup(cfg.LogLevel, cfg.LogFormat); err != nil {
		log.Fatal(err)
	}

	// Database schema is managed via SQL migrations (go run ./cmd/migrate), not GORM AutoMigrate
	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		slog.Error("connect to database", "error", err)
		os.Exit(1)
	}
	defer database.Close(db)

	router := routes.New(app.New(cfg, db))

	slog.Info("server starting", "port", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, router); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}
