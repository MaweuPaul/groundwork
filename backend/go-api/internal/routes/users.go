package routes

import (
	"net/http"

	"groundwork/internal/app"
)

// Protected: Authenticate has already run (see routes.go).
func registerUsers(mux *http.ServeMux, a *app.App) {
	mux.HandleFunc("GET /me", a.Users.Me)
}
