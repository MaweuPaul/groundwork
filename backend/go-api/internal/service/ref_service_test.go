package service

import (
	"testing"

	"github.com/google/uuid"
)

type fakeRefRepo struct {
	parcels map[string]uuid.UUID
	calls   int
}

func (f *fakeRefRepo) ParcelIDByShortID(shortID string) (uuid.UUID, bool, error) {
	f.calls++
	id, ok := f.parcels[shortID]
	return id, ok, nil
}

func (f *fakeRefRepo) SnapshotIDByShortID(string) (uuid.UUID, bool, error) {
	f.calls++
	return uuid.Nil, false, nil
}

func TestRefServiceParcelID(t *testing.T) {
	known := uuid.New()
	repo := &fakeRefRepo{parcels: map[string]uuid.UUID{"k7x2p9qd": known}}
	refs := NewRefService(repo)

	tests := []struct {
		name      string
		ref       string
		wantID    uuid.UUID
		wantFound bool
		wantCalls int
	}{
		{"uuid is used as is", known.String(), known, true, 0},
		{"known short id", "k7x2p9qd", known, true, 1},
		{"unknown short id", "k7x2p9qe", uuid.Nil, false, 1},
		{"uppercase is not a short id", "K7X2P9QD", uuid.Nil, false, 0},
		{"excluded letters are not a short id", "k7x2p9qi", uuid.Nil, false, 0},
		{"wrong length", "k7x2p9q", uuid.Nil, false, 0},
		{"slug is not a short id", "nyeri-block-a", uuid.Nil, false, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo.calls = 0
			id, found, err := refs.ParcelID(tt.ref)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if id != tt.wantID || found != tt.wantFound {
				t.Errorf("got (%v, %v), want (%v, %v)", id, found, tt.wantID, tt.wantFound)
			}
			if repo.calls != tt.wantCalls {
				t.Errorf("repository called %d times, want %d", repo.calls, tt.wantCalls)
			}
		})
	}
}
