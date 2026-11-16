package integration

import (
	"testing"
	"time"

	"gorm.io/gorm"

	"groundwork/internal/repository"
)

func TestSeriesSealsWhenEveryWindowFinishes(t *testing.T) {
	tx := begin(t)
	owner := newUser(t, tx, "asset_holder")
	parcel := newParcel(t, tx, owner, "Series parcel")
	m := ndvi(t, tx)

	ref, err := repository.NewSnapshotRepository(tx).CreateSeriesWithJobs(repository.NewSeries{
		Template: repository.NewSnapshot{
			ParcelID: parcel.ID, OwnerID: owner, MethodologyID: m.ID, MethodologyVersion: m.Version,
			Parameters: repository.Parameters{"min_clear_pct": 30},
		},
		DateStart:      day("2026-07-01"),
		DateEnd:        day("2026-08-31"),
		WindowMonths:   1,
		IntervalMonths: 1,
		Windows: [][2]time.Time{
			{day("2026-07-01"), day("2026-07-31")},
			{day("2026-08-01"), day("2026-08-31")},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if ref.Windows != 2 {
		t.Fatalf("windows = %d, want 2", ref.Windows)
	}

	sealed := func() *string {
		var hash *string
		if err := tx.Raw(`SELECT record_hash FROM snapshot_series WHERE id = ?`, ref.ID).Scan(&hash).Error; err != nil {
			t.Fatal(err)
		}
		return hash
	}

	first := claim(t, tx, worker)
	complete(t, tx, first, clearResult)
	if sealed() != nil {
		t.Fatal("series sealed with a window still queued")
	}

	// A failed window still finishes the series; its record says so.
	second := claim(t, tx, worker)
	if err := tx.Exec(`SELECT fail_snapshot_job(?, ?, 'NO_IMAGERY', 'none', false)`, second.JobID, worker).Error; err != nil {
		t.Fatal(err)
	}
	hash := sealed()
	if hash == nil {
		t.Fatal("series not sealed after its last window finished")
	}

	var row struct {
		Recomputed   string
		Text         string
		RecordFormat int
	}
	if err := tx.Raw(`
		SELECT series_record_hash(id, sealed_at) AS recomputed, series_record_text(id, sealed_at) AS text, record_format
		FROM snapshot_series WHERE id = ?`, ref.ID).Scan(&row).Error; err != nil {
		t.Fatal(err)
	}
	if row.Recomputed != *hash || row.RecordFormat != 2 {
		t.Errorf("recomputed %s, stored %s, format %d", row.Recomputed, *hash, row.RecordFormat)
	}

	expectError(t, tx, "Sealed series cannot be changed", func(tx *gorm.DB) error {
		return tx.Exec(`UPDATE snapshot_series SET window_months = 2 WHERE id = ?`, ref.ID).Error
	})
}
