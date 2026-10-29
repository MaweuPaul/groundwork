package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"groundwork/internal/models"
	"groundwork/internal/service"
)

var testSecret = []byte("test-secret-that-is-at-least-32-characters-long")

// fakeRefreshRepo satisfies RefreshTokenRepository so a real TokenService can issue tokens.
type fakeRefreshRepo struct{}

func (fakeRefreshRepo) Create(*models.RefreshToken) error              { return nil }
func (fakeRefreshRepo) GetByHash(string) (*models.RefreshToken, error) { return nil, nil }
func (fakeRefreshRepo) Revoke(uuid.UUID) error                         { return nil }

// fakeUserRepo returns users from a map, or err when set.
type fakeUserRepo struct {
	users map[uuid.UUID]*models.User
	err   error
}

func (f *fakeUserRepo) Create(*models.User) error                  { return nil }
func (f *fakeUserRepo) GetByEmail(string) (*models.User, error)    { return nil, nil }
func (f *fakeUserRepo) GetByUsername(string) (*models.User, error) { return nil, nil }
func (f *fakeUserRepo) GetByID(id uuid.UUID) (*models.User, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.users[id], nil
}

func newTokens(accessTTL time.Duration) service.TokenService {
	return service.NewTokenService(fakeRefreshRepo{}, testSecret, accessTTL, time.Hour)
}

func issueAccess(t *testing.T, tokens service.TokenService, userID uuid.UUID) string {
	t.Helper()
	pair, err := tokens.Issue(userID)
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}
	return pair.AccessToken
}

// recordingHandler notes whether it ran and which user it saw.
type recordingHandler struct {
	called bool
	user   *models.User
}

func (h *recordingHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.called = true
	h.user = UserFrom(r.Context())
	w.WriteHeader(http.StatusOK)
}

func assertError(t *testing.T, rec *httptest.ResponseRecorder, status int, message string) {
	t.Helper()
	if rec.Code != status {
		t.Fatalf("status = %d, want %d", rec.Code, status)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["error"] != message {
		t.Errorf("error = %q, want %q", body["error"], message)
	}
}

func TestAuthenticate(t *testing.T) {
	tokens := newTokens(15 * time.Minute)

	user := &models.User{ID: uuid.New(), Role: models.RoleAssetHolder}
	deletedID := uuid.New()
	users := &fakeUserRepo{users: map[uuid.UUID]*models.User{user.ID: user}}

	validToken := issueAccess(t, tokens, user.ID)
	deletedUserToken := issueAccess(t, tokens, deletedID)
	expiredToken := issueAccess(t, newTokens(-time.Minute), user.ID)

	otherSecret := service.NewTokenService(fakeRefreshRepo{}, []byte("a-completely-different-secret-of-32+-chars"), time.Hour, time.Hour)
	forgedToken := issueAccess(t, otherSecret, user.ID)

	noneToken, err := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.RegisteredClaims{
		Subject:   user.ID.String(),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	}).SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("sign none token: %v", err)
	}

	tests := []struct {
		name   string
		header string
	}{
		{"missing header", ""},
		{"wrong scheme", "Basic " + validToken},
		{"lowercase bearer", "bearer " + validToken},
		{"empty token", "Bearer "},
		{"garbage token", "Bearer not-a-jwt"},
		{"expired token", "Bearer " + expiredToken},
		{"signed with another secret", "Bearer " + forgedToken},
		{"alg none", "Bearer " + noneToken},
		{"user deleted", "Bearer " + deletedUserToken},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			next := &recordingHandler{}
			req := httptest.NewRequest(http.MethodGet, "/parcels", nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}
			rec := httptest.NewRecorder()

			NewAuthMiddleware(tokens, users).Authenticate(next).ServeHTTP(rec, req)

			assertError(t, rec, http.StatusUnauthorized, "unauthorized")
			if next.called {
				t.Error("next handler ran for a rejected request")
			}
		})
	}

	t.Run("valid token passes the user to next", func(t *testing.T) {
		next := &recordingHandler{}
		req := httptest.NewRequest(http.MethodGet, "/parcels", nil)
		req.Header.Set("Authorization", "Bearer "+validToken)
		rec := httptest.NewRecorder()

		NewAuthMiddleware(tokens, users).Authenticate(next).ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
		if !next.called {
			t.Fatal("next handler did not run")
		}
		if next.user != user {
			t.Errorf("UserFrom = %v, want %v", next.user, user)
		}
	})

	t.Run("database error is 500", func(t *testing.T) {
		next := &recordingHandler{}
		broken := &fakeUserRepo{err: errors.New("connection refused")}
		req := httptest.NewRequest(http.MethodGet, "/parcels", nil)
		req.Header.Set("Authorization", "Bearer "+validToken)
		rec := httptest.NewRecorder()

		NewAuthMiddleware(tokens, broken).Authenticate(next).ServeHTTP(rec, req)

		assertError(t, rec, http.StatusInternalServerError, "internal server error")
		if next.called {
			t.Error("next handler ran after a database error")
		}
	})
}

func TestUserFrom(t *testing.T) {
	if got := UserFrom(context.Background()); got != nil {
		t.Errorf("UserFrom on empty context = %v, want nil", got)
	}

	user := &models.User{ID: uuid.New()}
	ctx := context.WithValue(context.Background(), userKey{}, user)
	if got := UserFrom(ctx); got != user {
		t.Errorf("UserFrom = %v, want %v", got, user)
	}
}

func TestRequireRole(t *testing.T) {
	holder := &models.User{ID: uuid.New(), Role: models.RoleAssetHolder}
	auditor := &models.User{ID: uuid.New(), Role: models.RoleAuditor}

	tests := []struct {
		name     string
		user     *models.User
		wantPass bool
	}{
		{"matching role", holder, true},
		{"other role", auditor, false},
		{"no user in context", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			next := &recordingHandler{}
			req := httptest.NewRequest(http.MethodPost, "/parcels", nil)
			if tt.user != nil {
				req = req.WithContext(context.WithValue(req.Context(), userKey{}, tt.user))
			}
			rec := httptest.NewRecorder()

			RequireRole(models.RoleAssetHolder, next).ServeHTTP(rec, req)

			if tt.wantPass {
				if rec.Code != http.StatusOK || !next.called {
					t.Fatalf("status = %d, next called = %v; want 200 and called", rec.Code, next.called)
				}
				return
			}
			assertError(t, rec, http.StatusForbidden, "forbidden")
			if next.called {
				t.Error("next handler ran for the wrong role")
			}
		})
	}
}

// Routes chain both: Authenticate runs first, then RequireRole reads the user it stored.
func TestAuthenticateThenRequireRole(t *testing.T) {
	tokens := newTokens(15 * time.Minute)
	holder := &models.User{ID: uuid.New(), Role: models.RoleAssetHolder}
	auditor := &models.User{ID: uuid.New(), Role: models.RoleAuditor}
	users := &fakeUserRepo{users: map[uuid.UUID]*models.User{holder.ID: holder, auditor.ID: auditor}}
	auth := NewAuthMiddleware(tokens, users)

	tests := []struct {
		name       string
		user       *models.User
		wantStatus int
	}{
		{"asset holder creates parcel", holder, http.StatusOK},
		{"auditor is forbidden", auditor, http.StatusForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			next := &recordingHandler{}
			req := httptest.NewRequest(http.MethodPost, "/parcels", nil)
			req.Header.Set("Authorization", "Bearer "+issueAccess(t, tokens, tt.user.ID))
			rec := httptest.NewRecorder()

			auth.Authenticate(RequireRole(models.RoleAssetHolder, next)).ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}
