package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"time"

	"groundwork/internal/models"
	"groundwork/internal/repository"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

var ErrInvalidRefreshToken = errors.New("invalid refresh token")
var ErrInvalidAccessToken = errors.New("invalid access token")

type TokenService interface {
	Issue(userID uuid.UUID) (*TokenPair, error)
	Rotate(rawRefreshToken string) (*TokenPair, error)
	ParseAccess(accessToken string) (uuid.UUID, error) // middleware: "who is this?"
}

type tokenService struct {
	repo       repository.RefreshTokenRepository
	secret     []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
}

func NewTokenService(repo repository.RefreshTokenRepository, secret []byte, accessTTL, refreshTTL time.Duration) TokenService {
	return &tokenService{
		repo:       repo,
		secret:     secret,
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
	}
}

// Issue creates a new access token and refresh token for a user.
// Only the refresh token's hash is stored; the raw value goes to the client.
func (s *tokenService) Issue(userID uuid.UUID) (*TokenPair, error) {
	now := time.Now()

	access, err := s.signAccess(userID, now)
	if err != nil {
		return nil, err
	}

	raw, err := newRefreshToken()
	if err != nil {
		return nil, err
	}

	refresh := &models.RefreshToken{
		UserID:    userID,
		TokenHash: hashToken(raw),
		ExpiresAt: now.Add(s.refreshTTL),
	}
	if err := s.repo.Create(refresh); err != nil {
		return nil, err
	}

	return &TokenPair{AccessToken: access, RefreshToken: raw}, nil
}

// Rotate swaps a valid refresh token for a new pair and revokes the old one.
func (s *tokenService) Rotate(rawRefreshToken string) (*TokenPair, error) {
	stored, err := s.repo.GetByHash(hashToken(rawRefreshToken))
	if err != nil {
		return nil, err
	}
	if stored == nil || stored.RevokedAt != nil || time.Now().After(stored.ExpiresAt) {
		return nil, ErrInvalidRefreshToken
	}

	if err := s.repo.Revoke(stored.ID); err != nil {
		return nil, err
	}

	return s.Issue(stored.UserID)
}

// ParseAccess verifies an access token and returns the user ID inside it.
func (s *tokenService) ParseAccess(accessToken string) (uuid.UUID, error) {
	token, err := jwt.ParseWithClaims(accessToken, &jwt.RegisteredClaims{},
		func(t *jwt.Token) (any, error) { return s.secret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
	)
	if err != nil || !token.Valid {
		return uuid.Nil, ErrInvalidAccessToken
	}

	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok {
		return uuid.Nil, ErrInvalidAccessToken
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return uuid.Nil, ErrInvalidAccessToken
	}
	return userID, nil
}

func (s *tokenService) signAccess(userID uuid.UUID, now time.Time) (string, error) {
	claims := jwt.RegisteredClaims{
		Subject:   userID.String(),
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(s.accessTTL)),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.secret)
}

// newRefreshToken returns 32 random bytes as a URL-safe string.
func newRefreshToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
