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
	mux.HandleFunc("GET /parcels/{id}/snapshots", a.Snapshots.ListForParcel)
	mux.Handle("POST /parcels/{id}/series", middleware.RequireRole(models.RoleAssetHolder, http.HandlerFunc(a.Snapshots.CreateSeries)))
	mux.HandleFunc("GET /parcels/{id}/series", a.Snapshots.ListSeriesForParcel)
	mux.Handle("POST /parcels/{id}/groups", middleware.RequireRole(models.RoleAssetHolder, http.HandlerFunc(a.Snapshots.CreateGroup)))
	mux.HandleFunc("GET /parcels/{id}/groups", a.Snapshots.ListGroupsForParcel)
	mux.HandleFunc("GET /parcels/{id}/verify", a.Snapshots.VerifyParcel)
	mux.HandleFunc("GET /snapshots/{id}", a.Snapshots.Get)
	mux.HandleFunc("GET /snapshots/{id}/ndvi.png", a.Snapshots.NdviImage)
	mux.HandleFunc("GET /snapshots/{id}/artifacts/{key}", a.Snapshots.Artifact)
	mux.HandleFunc("GET /methodologies", a.Snapshots.ListMethodologies)
}
