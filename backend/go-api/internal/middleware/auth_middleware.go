package middleware

import (
	"context"
	"net/http"
	"strings"

	"groundwork/internal/httpx"
	"groundwork/internal/logging"
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
			httpx.WriteError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		// check if it starts with "Bearer "
		if !strings.HasPrefix(authHeader, "Bearer ") {
			httpx.WriteError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		// extract the token
		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == "" {
			httpx.WriteError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		// check if the token is genuine and still fresh
		userId, err := m.tokens.ParseAccess(token)
		if err != nil {
			httpx.WriteError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		//  check if the user still exists
		user, err := m.users.GetByID(userId)
		if err != nil {
			logging.From(r.Context()).Error("auth middleware: load user", "error", err)
			httpx.WriteError(w, http.StatusInternalServerError, "something went wrong")
			return
		}
		if user == nil {
			httpx.WriteError(w, http.StatusUnauthorized, "unauthorized") // deleted since token was issued
			return
		}

		// all good: attach the user and continue
		ctx := context.WithValue(logging.With(r.Context(), "user_id", user.ID), userKey{}, user)
		next.ServeHTTP(w, r.WithContext(ctx))

	})
}
