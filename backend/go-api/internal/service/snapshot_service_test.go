package service

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"groundwork/internal/models"
	"groundwork/internal/repository"

	"github.com/google/uuid"
)

type fakeSnapshotRepo struct {
	got       *repository.NewSnapshot
	gotSeries *repository.NewSeries
	id        uuid.UUID
	err       error
	calls     int
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

func (f *fakeSnapshotRepo) ArtifactForOwner(id, ownerID uuid.UUID, key string) (*repository.Artifact, error) {
	return nil, nil
}

func (f *fakeSnapshotRepo) CreateSeriesWithJobs(in repository.NewSeries) (repository.SeriesRef, error) {
	f.calls++
	f.gotSeries = &in
	return repository.SeriesRef{ID: f.id, Windows: len(in.Windows)}, f.err
}

func (f *fakeSnapshotRepo) ListSeriesByParcelForOwner(parcelID, ownerID uuid.UUID) ([]repository.SeriesSummary, error) {
	return nil, nil
}

type fakeMethodologyRepo struct {
	m      *models.Methodology
	params []models.MethodologyParameter
	err    error
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

func (f *fakeMethodologyRepo) Parameters(methodologyID uuid.UUID) ([]models.MethodologyParameter, error) {
	if f.params == nil {
		return ndviParams, nil
	}
	return f.params, nil
}

var ndvi = &models.Methodology{ID: uuid.New(), Name: "NDVI", Slug: "ndvi-s2-median", Version: "1.0.0", MaxCloudPct: 20,
	DataStart: time.Date(2015, 6, 23, 0, 0, 0, 0, time.UTC)}

var chirps = &models.Methodology{ID: uuid.New(), Name: "Seasonal rainfall, CHIRPS", Slug: "chirps-rain-season", Version: "1.0.0",
	DataStart: time.Date(1981, 1, 1, 0, 0, 0, 0, time.UTC)}

var ndviParams = []models.MethodologyParameter{
	{Name: "cloud_score_threshold", Label: "Clear pixel threshold", Type: "number", Min: 0.40, Max: 0.80, Default: 0.60},
	{Name: "max_scene_cloud_pct", Label: "Max scene cloud (%)", Type: "number", Min: 0.01, Max: 100, Default: 20},
	{Name: "min_clear_pct", Label: "Minimum clear pixels (%)", Type: "number", Min: 0, Max: 100, Default: 50},
}

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

func withParam(name string, value float64) func(*CreateSnapshotInput) {
	return func(in *CreateSnapshotInput) { in.Parameters = map[string]float64{name: value} }
}

func TestCreateSnapshotParameters(t *testing.T) {
	tests := []struct {
		name  string
		given map[string]float64
		want  repository.Parameters
	}{
		{"defaults from the methodology", nil,
			repository.Parameters{"max_scene_cloud_pct": 20, "cloud_score_threshold": 0.60, "min_clear_pct": 50}},
		{"custom values", map[string]float64{"max_scene_cloud_pct": 60, "cloud_score_threshold": 0.5},
			repository.Parameters{"max_scene_cloud_pct": 60, "cloud_score_threshold": 0.5, "min_clear_pct": 50}},
		{"edges are allowed", map[string]float64{"max_scene_cloud_pct": 100, "cloud_score_threshold": 0.4, "min_clear_pct": 0},
			repository.Parameters{"max_scene_cloud_pct": 100, "cloud_score_threshold": 0.4, "min_clear_pct": 0}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snaps := &fakeSnapshotRepo{id: uuid.New()}
			svc := newTestSnapshotService(snaps, &fakeMethodologyRepo{m: ndvi})
			in := validInput()
			in.Parameters = tt.given

			if _, err := svc.CreateSnapshot(in); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(snaps.got.Parameters, tt.want) {
				t.Errorf("parameters = %v, want %v", snaps.got.Parameters, tt.want)
			}
		})
	}
}

func TestCreateSnapshotWithoutParameters(t *testing.T) {
	snaps := &fakeSnapshotRepo{id: uuid.New()}
	svc := newTestSnapshotService(snaps, &fakeMethodologyRepo{m: ndvi, params: []models.MethodologyParameter{}})
	if _, err := svc.CreateSnapshot(validInput()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(snaps.got.Parameters) != 0 {
		t.Errorf("parameters = %v, want none", snaps.got.Parameters)
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
		{"before the methodology's data starts", func(in *CreateSnapshotInput) { in.DateStart = "2015-06-22" }},
		{"longer than a year", func(in *CreateSnapshotInput) { in.DateStart, in.DateEnd = "2023-01-01", "2024-01-03" }},
		{"zero scene cloud", withParam("max_scene_cloud_pct", 0)},
		{"scene cloud over 100", withParam("max_scene_cloud_pct", 100.5)},
		{"threshold too lenient", withParam("cloud_score_threshold", 0.3)},
		{"threshold too strict", withParam("cloud_score_threshold", 0.85)},
		{"negative min clear", withParam("min_clear_pct", -1)},
		{"unknown parameter", withParam("rain_threshold_mm", 5)},
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

func TestCreateSnapshotChecksTheMethodologysOwnDataStart(t *testing.T) {
	in := validInput()
	in.Methodology, in.Parameters = "chirps-rain-season", nil
	in.DateStart, in.DateEnd = "2010-03-01", "2010-05-31"
	chirpsRepo := &fakeMethodologyRepo{m: chirps, params: []models.MethodologyParameter{}}

	if _, err := newTestSnapshotService(&fakeSnapshotRepo{}, chirpsRepo).CreateSnapshot(in); err != nil {
		t.Fatalf("2010 is within CHIRPS data, got %v", err)
	}

	in.DateStart, in.DateEnd = "1980-12-31", "1981-03-31"
	_, err := newTestSnapshotService(&fakeSnapshotRepo{}, chirpsRepo).CreateSnapshot(in)
	var invalid *ValidationError
	if !errors.As(err, &invalid) || !strings.Contains(invalid.Message, "1981-01-01") {
		t.Fatalf("err = %v, want a ValidationError naming 1981-01-01", err)
	}
}
