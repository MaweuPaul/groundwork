package main

import (
	"log"
	"net/http"

	"groundwork/internal/config"
	"groundwork/internal/database"
	"groundwork/internal/handler"
	"groundwork/internal/repository"
	"groundwork/internal/service"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize Database
	db, err := database.Connect(cfg.DBURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close(db)

	// Run auto-migrations (optional, usually done via tools, but requested in DB module)
	if err := database.AutoMigrate(db); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	// Initialize repositories, services, and handlers
	parcelRepo := repository.NewParcelRepository(db)
	parcelSvc := service.NewParcelService(parcelRepo)
	parcelHandler := handler.NewParcelHandler(parcelSvc)

	// Setup Routes
	mux := http.NewServeMux()
	mux.HandleFunc("GET /parcels", parcelHandler.List)
	mux.HandleFunc("POST /parcels", parcelHandler.Create)

	// Start server
	log.Printf("Server starting on port %s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
