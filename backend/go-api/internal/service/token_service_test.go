package service

import (
	"errors"
	"sync"
	"testing"
	"time"

	"groundwork/internal/models"

	"github.com/google/uuid"
)

// memTokenRepo keeps refresh tokens in memory, with the same revoke-once rule as the database.
type memTokenRepo struct {
	mu     sync.Mutex
	tokens map[string]*models.RefreshToken
}

func newMemTokenRepo() *memTokenRepo {
	return &memTokenRepo{tokens: map[string]*models.RefreshToken{}}
}

func (m *memTokenRepo) Create(t *models.RefreshToken) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	t.ID = uuid.New()
	copied := *t
	m.tokens[t.TokenHash] = &copied
	return nil
}

func (m *memTokenRepo) GetByHash(hash string) (*models.RefreshToken, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	t, ok := m.tokens[hash]
	if !ok {
		return nil, nil
	}
	copied := *t
	return &copied, nil
}

func (m *memTokenRepo) Revoke(id uuid.UUID) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, t := range m.tokens {
		if t.ID == id {
			if t.RevokedAt != nil {
				return false, nil
			}
			now := time.Now()
			reason := "rotated"
			t.RevokedAt, t.RevokeReason = &now, &reason
			return true, nil
		}
	}
	return false, nil
}

func (m *memTokenRepo) RevokeForLogout(id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, t := range m.tokens {
		if t.ID == id {
			if t.RevokedAt == nil {
				now := time.Now()
				t.RevokedAt = &now
			}
			reason := "logout"
			t.RevokeReason = &reason
		}
	}
	return nil
}

// revokedAgo backdates a token's revocation to simulate the passage of time.
func (m *memTokenRepo) revokedAgo(raw string, d time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	at := time.Now().Add(-d)
	m.tokens[hashToken(raw)].RevokedAt = &at
}

func newTestTokenService(repo *memTokenRepo, refreshTTL time.Duration) TokenService {
	return NewTokenService(repo, []byte("test-secret-that-is-at-least-32-chars"), time.Minute, refreshTTL)
}

func TestRotateIssuesNewPairForSameUser(t *testing.T) {
	repo := newMemTokenRepo()
	svc := newTestTokenService(repo, time.Hour)
	user := uuid.New()

	first, err := svc.Issue(user)
	if err != nil {
		t.Fatal(err)
	}
	second, err := svc.Rotate(first.RefreshToken)
	if err != nil {
		t.Fatalf("rotate: %v", err)
	}
	if second.RefreshToken == first.RefreshToken {
		t.Error("rotation returned the same refresh token")
	}
	got, err := svc.ParseAccess(second.AccessToken)
	if err != nil || got != user {
		t.Errorf("new access token is for %v (err %v), want %v", got, err, user)
	}
}

func TestRotateReuseWithinGraceWindowSucceeds(t *testing.T) {
	repo := newMemTokenRepo()
	svc := newTestTokenService(repo, time.Hour)
	pair, _ := svc.Issue(uuid.New())

	if _, err := svc.Rotate(pair.RefreshToken); err != nil {
		t.Fatalf("first rotate: %v", err)
	}
	// A parallel request arriving a moment later with the same token.
	if _, err := svc.Rotate(pair.RefreshToken); err != nil {
		t.Fatalf("reuse inside the grace window: %v", err)
	}
}

func TestRotateReuseAfterGraceWindowFails(t *testing.T) {
	repo := newMemTokenRepo()
	svc := newTestTokenService(repo, time.Hour)
	pair, _ := svc.Issue(uuid.New())

	if _, err := svc.Rotate(pair.RefreshToken); err != nil {
		t.Fatalf("first rotate: %v", err)
	}
	repo.revokedAgo(pair.RefreshToken, refreshReuseGrace+time.Second)

	if _, err := svc.Rotate(pair.RefreshToken); !errors.Is(err, ErrInvalidRefreshToken) {
		t.Fatalf("err = %v, want ErrInvalidRefreshToken", err)
	}
}

func TestRotateRejectsUnknownAndExpired(t *testing.T) {
	repo := newMemTokenRepo()

	if _, err := newTestTokenService(repo, time.Hour).Rotate("not-a-real-token"); !errors.Is(err, ErrInvalidRefreshToken) {
		t.Errorf("unknown token: err = %v, want ErrInvalidRefreshToken", err)
	}

	expiring := newTestTokenService(repo, -time.Minute) // issues already-expired refresh tokens
	pair, _ := expiring.Issue(uuid.New())
	if _, err := expiring.Rotate(pair.RefreshToken); !errors.Is(err, ErrInvalidRefreshToken) {
		t.Errorf("expired token: err = %v, want ErrInvalidRefreshToken", err)
	}
}

func TestRotateConcurrentRequestsAllSucceed(t *testing.T) {
	repo := newMemTokenRepo()
	svc := newTestTokenService(repo, time.Hour)
	pair, _ := svc.Issue(uuid.New())

	const n = 8
	var wg sync.WaitGroup
	errs := make(chan error, n)
	for range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := svc.Rotate(pair.RefreshToken)
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			t.Errorf("concurrent rotate failed: %v", err)
		}
	}
}

func TestRevokeEndsSessionImmediately(t *testing.T) {
	repo := newMemTokenRepo()
	svc := newTestTokenService(repo, time.Hour)
	pair, _ := svc.Issue(uuid.New())

	if err := svc.Revoke(pair.RefreshToken); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	if _, err := svc.Rotate(pair.RefreshToken); !errors.Is(err, ErrInvalidRefreshToken) {
		t.Fatalf("refresh after logout: err = %v, want ErrInvalidRefreshToken", err)
	}
}

func TestRevokeEndsReuseWindowOfRotatedToken(t *testing.T) {
	repo := newMemTokenRepo()
	svc := newTestTokenService(repo, time.Hour)
	pair, _ := svc.Issue(uuid.New())

	if _, err := svc.Rotate(pair.RefreshToken); err != nil {
		t.Fatalf("rotate: %v", err)
	}
	if err := svc.Revoke(pair.RefreshToken); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	if _, err := svc.Rotate(pair.RefreshToken); !errors.Is(err, ErrInvalidRefreshToken) {
		t.Fatalf("reuse after logout: err = %v, want ErrInvalidRefreshToken", err)
	}
}

func TestRevokeUnknownTokenSucceeds(t *testing.T) {
	svc := newTestTokenService(newMemTokenRepo(), time.Hour)
	if err := svc.Revoke("not-a-real-token"); err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
}
