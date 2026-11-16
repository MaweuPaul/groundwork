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
	registerUsers(protected, a)
	registerParcels(protected, a)
	registerSnapshots(protected, a)
	registerActivity(protected, a)

	root := http.NewServeMux()
	registerAuth(root, a)
	root.Handle("/", a.Auth.Authenticate(protected))

	// Recover wraps Logger so a panic anywhere, even in Logger, becomes a 500. RequestID goes
	// outside both so the panic line has the ID too.
	return middleware.RequestID(middleware.Recover(middleware.Logger(root)))
}
