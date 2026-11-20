package integration

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"groundwork/internal/repository"
	"groundwork/internal/service"
)

func authService(tx *gorm.DB, demoEmail string) service.AuthService {
	tokens := service.NewTokenService(repository.NewRefreshTokenRepository(tx),
		[]byte(strings.Repeat("k", 32)), time.Minute, time.Hour)
	return service.NewAuthService(repository.NewUserRepository(tx), tokens, demoEmail)
}

func TestDemoSignsInAsTheDemoAccount(t *testing.T) {
	tx := begin(t)
	id := newUser(t, tx, "asset_holder")
	var email string
	if err := tx.Raw(`SELECT email FROM users WHERE id = ?`, id).Scan(&email).Error; err != nil {
		t.Fatal(err)
	}

	auth := authService(tx, " "+strings.ToUpper(email)+" ")
	if ok, err := auth.DemoAvailable(); err != nil || !ok {
		t.Fatalf("DemoAvailable = %v, %v; want true", ok, err)
	}
	user, pair, err := auth.Demo()
	if err != nil {
		t.Fatalf("Demo: %v", err)
	}
	if user.ID != id || pair.AccessToken == "" || pair.RefreshToken == "" {
		t.Fatalf("Demo signed in as %s with tokens %v; want %s", user.ID, pair, id)
	}
	if _, err := auth.Refresh(pair.RefreshToken); err != nil {
		t.Fatalf("the demo session's refresh token doesn't work: %v", err)
	}
}

func TestDemoIsOffWithoutAnAccount(t *testing.T) {
	tx := begin(t)
	for name, email := range map[string]string{"not configured": "", "no such account": uuid.NewString() + "@example.test"} {
		auth := authService(tx, email)
		if ok, err := auth.DemoAvailable(); err != nil || ok {
			t.Errorf("%s: DemoAvailable = %v, %v; want false", name, ok, err)
		}
		if _, _, err := auth.Demo(); !errors.Is(err, service.ErrDemoUnavailable) {
			t.Errorf("%s: Demo error = %v; want ErrDemoUnavailable", name, err)
		}
	}
}
