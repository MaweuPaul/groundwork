package app

import (
	"net/http"
	"time"

	"gorm.io/gorm"

	"groundwork/internal/config"
	"groundwork/internal/handler"
	"groundwork/internal/middleware"
	"groundwork/internal/ratelimit"
	"groundwork/internal/repository"
	"groundwork/internal/service"
)

// App holds everything the routes need, built once at start-up.
type App struct {
	Auth      *middleware.AuthMiddleware
	Users     *handler.AuthHandler
	Parcels   *handler.ParcelHandler
	Snapshots *handler.SnapshotHandler
	Activity  *handler.ActivityHandler
	Limits    *AuthLimits
}

// AuthLimits slow down password guessing and mass sign-ups. Login is limited per caller and
// per account, so guesses spread over many addresses still hit the account's limit.
type AuthLimits struct {
	ClientIP   func(*http.Request) string
	LoginIP    *ratelimit.Limiter
	LoginEmail *ratelimit.Limiter
	Register   *ratelimit.Limiter
}

// New wires repositories into services, and services into handlers.
func New(cfg *config.Config, db *gorm.DB) *App {
	// Auth
	userRepo := repository.NewUserRepository(db)
	tokenRepo := repository.NewRefreshTokenRepository(db)
	tokenSvc := service.NewTokenService(tokenRepo, []byte(cfg.JWTSecret), cfg.AccessTokenTTL, cfg.RefreshTokenTTL)
	authSvc := service.NewAuthService(userRepo, tokenSvc, cfg.DemoEmail)

	// URL ids: UUID or short ID
	refSvc := service.NewRefService(repository.NewRefRepository(db))

	// Parcels
	parcelRepo := repository.NewParcelRepository(db)
	parcelSvc := service.NewParcelService(parcelRepo)

	// Snapshots
	snapshotRepo := repository.NewSnapshotRepository(db)
	methodologyRepo := repository.NewMethodologyRepository(db)
	snapshotSvc := service.NewSnapshotService(snapshotRepo, methodologyRepo, repository.Limits{
		PerDay:    cfg.SnapshotsPerDay,
		MaxAreaHa: cfg.MaxSnapshotAreaHa,
	})

	activitySvc := service.NewActivityService(repository.NewActivityRepository(db))

	return &App{
		Auth:      middleware.NewAuthMiddleware(tokenSvc, userRepo),
		Users:     handler.NewAuthHandler(authSvc),
		Parcels:   handler.NewParcelHandler(parcelSvc, refSvc),
		Snapshots: handler.NewSnapshotHandler(snapshotSvc, refSvc),
		Activity:  handler.NewActivityHandler(activitySvc),
		Limits: &AuthLimits{
			ClientIP:   middleware.ClientIP(cfg.ProxySecret, cfg.BehindProxy),
			LoginIP:    ratelimit.New(10, time.Minute),
			LoginEmail: ratelimit.New(5, 3*time.Minute),
			Register:   ratelimit.New(3, 20*time.Minute),
		},
	}
}
