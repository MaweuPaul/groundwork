package routes

import (
	"net/http"

	"groundwork/internal/app"
	"groundwork/internal/middleware"
)

// New builds the API router. Only routes registered on root are public;
// everything else goes through Authenticate.
func New(a *app.App) http.Handler {
	protected := http.NewServeMux()
	registerParcels(protected, a)

	root := http.NewServeMux()
	registerAuth(root, a)
	root.Handle("/", a.Auth.Authenticate(protected))

	// Recover is outermost so a panic anywhere, even in Logger, becomes a 500.
	return middleware.Recover(middleware.Logger(root))
}
