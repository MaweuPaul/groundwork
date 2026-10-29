package app

import (
	"gorm.io/gorm"

	"groundwork/internal/config"
	"groundwork/internal/handler"
	"groundwork/internal/middleware"
	"groundwork/internal/repository"
	"groundwork/internal/service"
)

// App holds everything the routes need, built once at start-up.
type App struct {
	Auth    *middleware.AuthMiddleware
	Users   *handler.AuthHandler
	Parcels *handler.ParcelHandler
}

// New wires repositories into services, and services into handlers.
func New(cfg *config.Config, db *gorm.DB) *App {
	// Auth
	userRepo := repository.NewUserRepository(db)
	tokenRepo := repository.NewRefreshTokenRepository(db)
	tokenSvc := service.NewTokenService(tokenRepo, []byte(cfg.JWTSecret), cfg.AccessTokenTTL, cfg.RefreshTokenTTL)
	authSvc := service.NewAuthService(userRepo, tokenSvc)

	// Parcels
	parcelRepo := repository.NewParcelRepository(db)
	parcelSvc := service.NewParcelService(parcelRepo)

	// The GEE client (service.NewGEEServiceClient(cfg.GEEServiceURL)) is added here
	// once the snapshot service uses it.

	return &App{
		Auth:    middleware.NewAuthMiddleware(tokenSvc, userRepo),
		Users:   handler.NewAuthHandler(authSvc),
		Parcels: handler.NewParcelHandler(parcelSvc),
	}
}
