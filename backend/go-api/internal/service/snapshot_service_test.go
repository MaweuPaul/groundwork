package service

import (
	"errors"
	"testing"
	"time"

	"groundwork/internal/models"
	"groundwork/internal/repository"

	"github.com/google/uuid"
)

type fakeSnapshotRepo struct {
	got   *repository.NewSnapshot
	id    uuid.UUID
	err   error
	calls int
}

func (f *fakeSnapshotRepo) CreateWithJob(in repository.NewSnapshot) (uuid.UUID, error) {
	f.calls++
	f.got = &in
	return f.id, f.err
}

func (f *fakeSnapshotRepo) GetByIDForOwner(id, ownerID uuid.UUID) (*repository.SnapshotDetail, error) {
	return nil, nil
}

func (f *fakeSnapshotRepo) ListByParcelForOwner(parcelID, ownerID uuid.UUID) ([]repository.SnapshotSummary, error) {
	return nil, nil
}

func (f *fakeSnapshotRepo) NdviPNGForOwner(id, ownerID uuid.UUID) ([]byte, error) {
	return nil, nil
}

type fakeMethodologyRepo struct {
	m   *models.Methodology
	err error
}

func (f *fakeMethodologyRepo) GetBySlugVersion(slug, version string) (*models.Methodology, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.m != nil && f.m.Slug == slug && f.m.Version == version {
		return f.m, nil
	}
	return nil, nil
}

func (f *fakeMethodologyRepo) List() ([]models.Methodology, error) {
	return nil, nil
}

var ndvi = &models.Methodology{ID: uuid.New(), Slug: "ndvi-s2-median", Version: "1.0.0", MaxCloudPct: 20}

// newTestSnapshotService fixes "now" at 2026-11-02 so date rules are deterministic.
func newTestSnapshotService(snaps *fakeSnapshotRepo, meths *fakeMethodologyRepo) *snapshotService {
	return &snapshotService{
		snapshots:     snaps,
		methodologies: meths,
		now:           func() time.Time { return time.Date(2026, 11, 2, 9, 0, 0, 0, time.UTC) },
	}
}

func validInput() CreateSnapshotInput {
	return CreateSnapshotInput{
		ParcelID:           uuid.New(),
		OwnerID:            uuid.New(),
		Methodology:        "ndvi-s2-median",
		MethodologyVersion: "1.0.0",
		DateStart:          "2025-03-01",
		DateEnd:            "2025-04-30",
	}
}

func TestCreateSnapshotQueuesValidRequest(t *testing.T) {
	snaps := &fakeSnapshotRepo{id: uuid.New()}
	svc := newTestSnapshotService(snaps, &fakeMethodologyRepo{m: ndvi})
	in := validInput()

	id, err := svc.CreateSnapshot(in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != snaps.id {
		t.Errorf("id = %v, want %v", id, snaps.id)
	}

	got := snaps.got
	if got.ParcelID != in.ParcelID || got.OwnerID != in.OwnerID {
		t.Errorf("parcel/owner not passed through: %+v", got)
	}
	if got.MethodologyID != ndvi.ID || got.MethodologyVersion != "1.0.0" {
		t.Errorf("methodology = %v@%s, want %v@1.0.0", got.MethodologyID, got.MethodologyVersion, ndvi.ID)
	}
	if got.DateStart.Format(time.DateOnly) != "2025-03-01" || got.DateEnd.Format(time.DateOnly) != "2025-04-30" {
		t.Errorf("dates = %s..%s", got.DateStart.Format(time.DateOnly), got.DateEnd.Format(time.DateOnly))
	}
}

func ptr(v float64) *float64 { return &v }

func TestCreateSnapshotCloudOptions(t *testing.T) {
	tests := []struct {
		name          string
		cloudPct      *float64
		threshold     *float64
		wantCloudPct  float64
		wantThreshold float64
	}{
		{"defaults from the methodology", nil, nil, 20, 0.60},
		{"custom values", ptr(60), ptr(0.5), 60, 0.5},
		{"rounded like the database stores them", ptr(33.333), ptr(0.6256), 33.33, 0.626},
		{"edges are allowed", ptr(100), ptr(0.4), 100, 0.4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snaps := &fakeSnapshotRepo{id: uuid.New()}
			svc := newTestSnapshotService(snaps, &fakeMethodologyRepo{m: ndvi})
			in := validInput()
			in.MaxSceneCloudPct, in.CloudScoreThreshold = tt.cloudPct, tt.threshold

			if _, err := svc.CreateSnapshot(in); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if snaps.got.MaxSceneCloudPct != tt.wantCloudPct || snaps.got.CloudScoreThreshold != tt.wantThreshold {
				t.Errorf("got %v%% / %v, want %v%% / %v",
					snaps.got.MaxSceneCloudPct, snaps.got.CloudScoreThreshold, tt.wantCloudPct, tt.wantThreshold)
			}
		})
	}
}

func TestCreateSnapshotRejectsBadInput(t *testing.T) {
	tests := []struct {
		name   string
		modify func(*CreateSnapshotInput)
	}{
		{"missing methodology", func(in *CreateSnapshotInput) { in.Methodology = " " }},
		{"missing version", func(in *CreateSnapshotInput) { in.MethodologyVersion = "" }},
		{"bad start format", func(in *CreateSnapshotInput) { in.DateStart = "01/03/2025" }},
		{"bad end format", func(in *CreateSnapshotInput) { in.DateEnd = "2025-13-01" }},
		{"end before start", func(in *CreateSnapshotInput) { in.DateStart, in.DateEnd = "2025-04-30", "2025-03-01" }},
		{"end in the future", func(in *CreateSnapshotInput) { in.DateEnd = "2026-11-03" }},
		{"before Sentinel-2", func(in *CreateSnapshotInput) { in.DateStart = "2015-06-22" }},
		{"longer than a year", func(in *CreateSnapshotInput) { in.DateStart, in.DateEnd = "2023-01-01", "2024-01-03" }},
		{"zero scene cloud", func(in *CreateSnapshotInput) { in.MaxSceneCloudPct = ptr(0) }},
		{"scene cloud over 100", func(in *CreateSnapshotInput) { in.MaxSceneCloudPct = ptr(100.5) }},
		{"threshold too lenient", func(in *CreateSnapshotInput) { in.CloudScoreThreshold = ptr(0.3) }},
		{"threshold too strict", func(in *CreateSnapshotInput) { in.CloudScoreThreshold = ptr(0.85) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snaps := &fakeSnapshotRepo{}
			svc := newTestSnapshotService(snaps, &fakeMethodologyRepo{m: ndvi})
			in := validInput()
			tt.modify(&in)

			_, err := svc.CreateSnapshot(in)
			var invalid *ValidationError
			if !errors.As(err, &invalid) {
				t.Fatalf("err = %v, want *ValidationError", err)
			}
			if snaps.calls != 0 {
				t.Error("repository was called for an invalid request")
			}
		})
	}
}

func TestCreateSnapshotAcceptsEdgeDates(t *testing.T) {
	tests := []struct{ start, end string }{
		{"2015-06-23", "2015-07-01"}, // first day of Sentinel-2
		{"2026-10-01", "2026-11-02"}, // ends today
		{"2025-03-01", "2025-03-01"}, // single day
	}
	for _, tt := range tests {
		svc := newTestSnapshotService(&fakeSnapshotRepo{}, &fakeMethodologyRepo{m: ndvi})
		in := validInput()
		in.DateStart, in.DateEnd = tt.start, tt.end
		if _, err := svc.CreateSnapshot(in); err != nil {
			t.Errorf("%s..%s: unexpected error: %v", tt.start, tt.end, err)
		}
	}
}

func TestCreateSnapshotUnknownMethodology(t *testing.T) {
	snaps := &fakeSnapshotRepo{}
	svc := newTestSnapshotService(snaps, &fakeMethodologyRepo{m: ndvi})
	in := validInput()
	in.MethodologyVersion = "9.9.9"

	if _, err := svc.CreateSnapshot(in); !errors.Is(err, ErrUnknownMethodology) {
		t.Fatalf("err = %v, want ErrUnknownMethodology", err)
	}
	if snaps.calls != 0 {
		t.Error("repository was called for an unknown methodology")
	}
}

func TestCreateSnapshotPassesRepositoryErrors(t *testing.T) {
	dbErr := errors.New("connection reset")
	tests := []struct {
		name  string
		snaps *fakeSnapshotRepo
		meths *fakeMethodologyRepo
		want  error
	}{
		{"parcel not found", &fakeSnapshotRepo{err: repository.ErrParcelNotFound}, &fakeMethodologyRepo{m: ndvi}, repository.ErrParcelNotFound},
		{"snapshot insert fails", &fakeSnapshotRepo{err: dbErr}, &fakeMethodologyRepo{m: ndvi}, dbErr},
		{"methodology lookup fails", &fakeSnapshotRepo{}, &fakeMethodologyRepo{err: dbErr}, dbErr},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newTestSnapshotService(tt.snaps, tt.meths)
			if _, err := svc.CreateSnapshot(validInput()); !errors.Is(err, tt.want) {
				t.Fatalf("err = %v, want %v", err, tt.want)
			}
		})
	}
}
