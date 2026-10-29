package middleware

import (
	"net/http"

	"groundwork/internal/httpx"
	"groundwork/internal/models"
)

func RequireRole(role models.UserRole, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := UserFrom(r.Context())
		if user == nil || user.Role != role {
			httpx.WriteError(w, http.StatusForbidden, "forbidden")
			return
		}
		next.ServeHTTP(w, r)
	})
}
