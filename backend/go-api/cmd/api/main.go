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
	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close(db)

	// Run auto-migrations (optional, usually done via tools, but requested in DB module)
	// Database schema is managed via SQL migrations, not GORM AutoMigrate

	// Initialize repositories, services, and handlers
	parcelRepo := repository.NewParcelRepository(db)
	parcelSvc := service.NewParcelService(parcelRepo)
	
	geeClient := service.NewGEEServiceClient(cfg.GEEServiceURL)
	_ = geeClient // Will be injected into a snapshot service later

	parcelHandler := handler.NewParcelHandler(parcelSvc)

	userRepo := repository.NewUserRepository(db)
	tokenRepo := repository.NewRefreshTokenRepository(db)
	tokenSvc := service.NewTokenService(tokenRepo, []byte(cfg.JWTSecret), cfg.AccessTokenTTL, cfg.RefreshTokenTTL)
	authSvc := service.NewAuthService(userRepo, tokenSvc)
	authHandler := handler.NewAuthHandler(authSvc)

	// Setup Routes
	mux := http.NewServeMux()
	mux.HandleFunc("GET /parcels", parcelHandler.List)
	mux.HandleFunc("POST /parcels", parcelHandler.Create)

	mux.HandleFunc("POST /auth/register", authHandler.Register)
	mux.HandleFunc("POST /auth/login", authHandler.Login)
	mux.HandleFunc("POST /auth/refresh", authHandler.Refresh)

	// Start server
	log.Printf("Server starting on port %s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
