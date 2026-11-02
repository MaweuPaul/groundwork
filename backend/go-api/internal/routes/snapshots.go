package routes

import (
	"net/http"

	"groundwork/internal/app"
	"groundwork/internal/middleware"
	"groundwork/internal/models"
)

// Protected: Authenticate has already run (see routes.go).
func registerSnapshots(mux *http.ServeMux, a *app.App) {
	mux.Handle("POST /parcels/{id}/snapshots", middleware.RequireRole(models.RoleAssetHolder, http.HandlerFunc(a.Snapshots.Create)))
}
