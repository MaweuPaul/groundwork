package routes

import (
	"net/http"

	"groundwork/internal/app"
	"groundwork/internal/middleware"
)

// Public: no token needed.
func registerAuth(mux *http.ServeMux, a *app.App) {
	l := a.Limits
	mux.Handle("POST /auth/register",
		middleware.RateLimit(l.Register, l.ClientIP, http.HandlerFunc(a.Users.Register)))
	mux.Handle("POST /auth/login",
		middleware.RateLimit(l.LoginIP, l.ClientIP,
			middleware.RateLimit(l.LoginEmail, middleware.EmailKey, http.HandlerFunc(a.Users.Login))))
	mux.HandleFunc("POST /auth/refresh", a.Users.Refresh)
	mux.HandleFunc("POST /auth/logout", a.Users.Logout)
}
