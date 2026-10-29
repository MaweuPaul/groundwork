package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"groundwork/internal/models"
	"groundwork/internal/repository"
	"groundwork/internal/service"
)

type userKey struct{}

func UserFrom(ctx context.Context) *models.User {
	user, _ := ctx.Value(userKey{}).(*models.User)
	return user
}

type AuthMiddleware struct {
	tokens service.TokenService
	users  repository.UserRepository
}

func NewAuthMiddleware(tokens service.TokenService, users repository.UserRepository) *AuthMiddleware {
	return &AuthMiddleware{tokens: tokens, users: users}
}

func (m *AuthMiddleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// get authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		// check if it starts with "Bearer "
		if !strings.HasPrefix(authHeader, "Bearer ") {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		// extract the token
		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == "" {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		// check if the token is genuine and still fresh
		userId, err := m.tokens.ParseAccess(token)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		//  check if the user still exists
		user, err := m.users.GetByID(userId)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal server error")
			return
		}
		if user == nil {
			writeError(w, http.StatusUnauthorized, "unauthorized") // deleted since token was issued
			return
		}

		// all good: attach the user and continue
		ctx := context.WithValue(r.Context(), userKey{}, user)
		next.ServeHTTP(w, r.WithContext(ctx))

	})
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
