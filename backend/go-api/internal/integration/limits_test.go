package integration

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"groundwork/internal/models"
	"groundwork/internal/repository"
)

func snapshotRequest(t *testing.T, tx *gorm.DB, owner uuid.UUID, parcel models.Parcel, start, end string, limits repository.Limits) repository.NewSnapshot {
	m := ndvi(t, tx)
	return repository.NewSnapshot{
		ParcelID: parcel.ID, OwnerID: owner, MethodologyID: m.ID, MethodologyVersion: m.Version,
		DateStart: day(start), DateEnd: day(end),
		Parameters: repository.Parameters{"min_clear_pct": 30},
		Limits:     limits,
	}
}

func limitError(t *testing.T, err error) *repository.LimitError {
	t.Helper()
	var limit *repository.LimitError
	if !errors.As(err, &limit) {
		t.Fatalf("got %v, want a LimitError", err)
	}
	return limit
}

func snapshotCount(t *testing.T, tx *gorm.DB, parcel models.Parcel) int {
	var n int
	if err := tx.Raw(`SELECT count(*) FROM snapshots WHERE parcel_id = ?`, parcel.ID).Scan(&n).Error; err != nil {
		t.Fatal(err)
	}
	return n
}

func TestDuplicateRequestReturnsQueuedSnapshot(t *testing.T) {
	tx := begin(t)
	owner := newUser(t, tx, "asset_holder")
	parcel := newParcel(t, tx, owner, "Double click")
	repo := repository.NewSnapshotRepository(tx)
	req := snapshotRequest(t, tx, owner, parcel, "2026-10-01", "2026-10-30", repository.Limits{})

	first, err := repo.CreateWithJob(req)
	if err != nil {
		t.Fatal(err)
	}
	second, err := repo.CreateWithJob(req)
	if err != nil {
		t.Fatal(err)
	}
	if first != second || snapshotCount(t, tx, parcel) != 1 {
		t.Fatalf("a repeated request queued another snapshot (%s, %s)", first, second)
	}

	other := req
	other.Parameters = repository.Parameters{"min_clear_pct": 40}
	if id, err := repo.CreateWithJob(other); err != nil || id == first {
		t.Fatalf("different parameters should queue a new snapshot: %v", err)
	}

	// Stored parameters have the defaults filled in; a request that leaves them out still matches.
	// Once the first has finished, the same request is a deliberate rerun.
	job := claim(t, tx, "integration-worker")
	for job != nil && job.SnapshotID != first {
		job = claim(t, tx, "integration-worker")
	}
	complete(t, tx, job, clearResult)
	if id, err := repo.CreateWithJob(req); err != nil || id == first {
		t.Fatalf("a rerun after completion should queue a new snapshot: %v", err)
	}
}

func TestDailyCap(t *testing.T) {
	tx := begin(t)
	owner := newUser(t, tx, "asset_holder")
	parcel := newParcel(t, tx, owner, "Capped")
	repo := repository.NewSnapshotRepository(tx)
	limits := repository.Limits{PerDay: 3}

	for i, month := range []string{"07", "08", "09"} {
		if _, err := repo.CreateWithJob(snapshotRequest(t, tx, owner, parcel, "2026-"+month+"-01", "2026-"+month+"-28", limits)); err != nil {
			t.Fatalf("snapshot %d within the cap: %v", i+1, err)
		}
	}

	_, err := repo.CreateWithJob(snapshotRequest(t, tx, owner, parcel, "2026-10-01", "2026-10-28", limits))
	limit := limitError(t, err)
	if !limit.Quota || !strings.Contains(limit.Message, "used all 3 snapshots") || !strings.Contains(limit.Message, "frees up in 24 hours") {
		t.Fatalf("unexpected refusal: %+v", limit)
	}

	// A duplicate of a queued one is still answered at the cap: it costs nothing.
	if _, err := repo.CreateWithJob(snapshotRequest(t, tx, owner, parcel, "2026-07-01", "2026-07-28", limits)); err != nil {
		t.Fatalf("duplicate at the cap: %v", err)
	}

	// Another owner has their own allowance.
	other := newUser(t, tx, "asset_holder")
	otherParcel := newParcel(t, tx, other, "Not capped")
	if _, err := repo.CreateWithJob(snapshotRequest(t, tx, other, otherParcel, "2026-10-01", "2026-10-28", limits)); err != nil {
		t.Fatalf("another owner was capped: %v", err)
	}

	// Snapshots older than a day no longer count.
	if err := tx.Exec(`ALTER TABLE snapshots DISABLE TRIGGER USER`).Error; err != nil {
		t.Fatal(err)
	}
	if err := tx.Exec(`UPDATE snapshots SET created_at = now() - interval '25 hours' WHERE parcel_id = ?`, parcel.ID).Error; err != nil {
		t.Fatal(err)
	}
	if err := tx.Exec(`ALTER TABLE snapshots ENABLE TRIGGER USER`).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := repo.CreateWithJob(snapshotRequest(t, tx, owner, parcel, "2026-10-01", "2026-10-28", limits)); err != nil {
		t.Fatalf("after 24 hours: %v", err)
	}
}

func TestSeriesCountsEveryWindow(t *testing.T) {
	tx := begin(t)
	owner := newUser(t, tx, "asset_holder")
	parcel := newParcel(t, tx, owner, "Series cap")
	repo := repository.NewSnapshotRepository(tx)
	limits := repository.Limits{PerDay: 4}

	if _, err := repo.CreateWithJob(snapshotRequest(t, tx, owner, parcel, "2026-01-01", "2026-01-28", limits)); err != nil {
		t.Fatal(err)
	}

	series := repository.NewSeries{
		Template:     snapshotRequest(t, tx, owner, parcel, "2026-07-01", "2026-10-31", limits),
		DateStart:    day("2026-07-01"),
		DateEnd:      day("2026-10-31"),
		WindowMonths: 1, IntervalMonths: 1,
		Windows: [][2]time.Time{
			{day("2026-07-01"), day("2026-07-31")}, {day("2026-08-01"), day("2026-08-31")},
			{day("2026-09-01"), day("2026-09-30")}, {day("2026-10-01"), day("2026-10-31")},
		},
	}
	_, err := repo.CreateSeriesWithJobs(series)
	limit := limitError(t, err)
	if !strings.Contains(limit.Message, "needs 4 snapshots, but only 3 of your 4") {
		t.Fatalf("unexpected refusal: %+v", limit)
	}

	series.Windows = series.Windows[:3]
	series.DateEnd = day("2026-09-30")
	first, err := repo.CreateSeriesWithJobs(series)
	if err != nil {
		t.Fatalf("a series that fits: %v", err)
	}
	again, err := repo.CreateSeriesWithJobs(series)
	if err != nil || again.ID != first.ID || again.Windows != 3 {
		t.Fatalf("repeated series: %+v, %v; want the queued one back", again, err)
	}
}

func TestAreaLimit(t *testing.T) {
	tx := begin(t)
	owner := newUser(t, tx, "asset_holder")
	parcel := newParcel(t, tx, owner, "Too big")
	var area float64
	if err := tx.Raw(`SELECT area_ha FROM parcels WHERE id = ?`, parcel.ID).Scan(&area).Error; err != nil {
		t.Fatal(err)
	}
	repo := repository.NewSnapshotRepository(tx)

	_, err := repo.CreateWithJob(snapshotRequest(t, tx, owner, parcel, "2026-10-01", "2026-10-28", repository.Limits{MaxAreaHa: area / 2}))
	limit := limitError(t, err)
	if limit.Quota || !strings.Contains(limit.Message, "Snapshots are limited to parcels of") {
		t.Fatalf("unexpected refusal: %+v", limit)
	}
	if snapshotCount(t, tx, parcel) != 0 {
		t.Fatal("a refused request left a snapshot behind")
	}

	if _, err := repo.CreateWithJob(snapshotRequest(t, tx, owner, parcel, "2026-10-01", "2026-10-28", repository.Limits{MaxAreaHa: area * 2})); err != nil {
		t.Fatalf("within the area limit: %v", err)
	}

	stranger := newUser(t, tx, "asset_holder")
	_, err = repo.CreateWithJob(snapshotRequest(t, tx, stranger, parcel, "2026-10-01", "2026-10-28", repository.Limits{}))
	if !errors.Is(err, repository.ErrParcelNotFound) {
		t.Fatalf("someone else's parcel: %v, want ErrParcelNotFound", err)
	}
}
