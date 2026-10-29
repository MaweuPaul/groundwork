package routes

import (
	"net/http"

	"groundwork/internal/app"
	"groundwork/internal/middleware"
	"groundwork/internal/models"
)

// Protected: Authenticate has already run (see routes.go).
func registerParcels(mux *http.ServeMux, a *app.App) {
	mux.HandleFunc("GET /parcels", a.Parcels.List)
	mux.Handle("POST /parcels", middleware.RequireRole(models.RoleAssetHolder, http.HandlerFunc(a.Parcels.Create)))
}
