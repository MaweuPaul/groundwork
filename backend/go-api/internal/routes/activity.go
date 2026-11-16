package routes

import (
	"net/http"

	"groundwork/internal/app"
)

// Protected: Authenticate has already run (see routes.go).
func registerActivity(mux *http.ServeMux, a *app.App) {
	mux.HandleFunc("GET /activity", a.Activity.List)
}
