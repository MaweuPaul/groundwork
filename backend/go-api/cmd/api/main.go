package main

import (
	"log"
	"net/http"

	"groundwork/internal/app"
	"groundwork/internal/config"
	"groundwork/internal/database"
	"groundwork/internal/routes"
)

func main() {
	cfg := config.Load()

	// Database schema is managed via SQL migrations (go run ./cmd/migrate), not GORM AutoMigrate
	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close(db)

	router := routes.New(app.New(cfg, db))

	log.Printf("Server starting on port %s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, router); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
