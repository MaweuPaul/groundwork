package middleware

import (
	"context"
	"log"
	"net/http"
	"strings"

	"groundwork/internal/httpx"
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
			log.Printf("auth middleware: load user: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, "something went wrong")
			return
		}
		if user == nil {
			httpx.WriteError(w, http.StatusUnauthorized, "unauthorized") // deleted since token was issued
			return
		}

		// all good: attach the user and continue
		ctx := context.WithValue(r.Context(), userKey{}, user)
		next.ServeHTTP(w, r.WithContext(ctx))

	})
}
