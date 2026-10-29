package routes

import (
	"net/http"

	"groundwork/internal/app"
)

// Public: no token needed.
func registerAuth(mux *http.ServeMux, a *app.App) {
	mux.HandleFunc("POST /auth/register", a.Users.Register)
	mux.HandleFunc("POST /auth/login", a.Users.Login)
	mux.HandleFunc("POST /auth/refresh", a.Users.Refresh)
}
