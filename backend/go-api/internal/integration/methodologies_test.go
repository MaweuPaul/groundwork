package integration

import (
	"crypto/sha256"
	"encoding/hex"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"

	"groundwork/internal/models"
	"groundwork/internal/repository"
)

func TestMethodologiesPublishMeasurementsManifestAndRecords(t *testing.T) {
	tx := begin(t)
	owner := newUser(t, tx, "asset_holder")
	other := newUser(t, tx, "asset_holder")
	runSnapshot(t, tx, owner, newParcel(t, tx, owner, "Methodologies parcel"), clearResult)

	list := func(ownerID uuid.UUID) map[string]models.Methodology {
		t.Helper()
		methodologies, err := repository.NewMethodologyRepository(tx).List(ownerID)
		if err != nil {
			t.Fatal(err)
		}
		bySlug := map[string]models.Methodology{}
		for _, m := range methodologies {
			bySlug[m.Slug] = m
		}
		return bySlug
	}
	mine, theirs := list(owner), list(other)

	want := map[string][]string{
		"ndvi-s2-median":     {"ndvi_mean", "ndvi_max", "valid_pixel_pct"},
		"chirps-rain-season": {"rain_total_mm", "rain_normal_mm", "rain_pct_of_normal", "rainy_days", "longest_dry_spell_days"},
	}
	for slug, measurements := range want {
		m, ok := mine[slug]
		if !ok {
			t.Fatalf("%s is not listed", slug)
		}
		var got []string
		for _, mm := range m.Measurements {
			got = append(got, mm.Slug)
			if mm.Name == "" {
				t.Errorf("%s: measurement %s has no name", slug, mm.Slug)
			}
		}
		if !slices.Equal(got, measurements) {
			t.Errorf("%s measurements = %v, want %v", slug, got, measurements)
		}

		sum := sha256.Sum256([]byte(m.ManifestText))
		if !strings.HasPrefix(m.ManifestText, "groundwork-manifest-v1\nslug="+slug+"\n") || m.ManifestSHA256 != hex.EncodeToString(sum[:]) {
			t.Errorf("%s manifest = %q, hash %s", slug, m.ManifestText, m.ManifestSHA256)
		}
	}

	if mine["ndvi-s2-median"].Records != 1 || mine["chirps-rain-season"].Records != 0 {
		t.Errorf("owner's records: ndvi %d, chirps %d; want 1, 0", mine["ndvi-s2-median"].Records, mine["chirps-rain-season"].Records)
	}
	if theirs["ndvi-s2-median"].Records != 0 {
		t.Errorf("another user's ndvi records = %d, want 0", theirs["ndvi-s2-median"].Records)
	}
}

func TestMethodologiesCarryTheLatestDayOfData(t *testing.T) {
	tx := begin(t)
	if err := tx.Exec(`
		INSERT INTO methodology_coverage (methodology_id, data_end)
		SELECT id, '2026-09-30' FROM methodologies WHERE slug = 'chirps-rain-season'`).Error; err != nil {
		t.Fatal(err)
	}
	repo := repository.NewMethodologyRepository(tx)

	chirps, err := repo.GetBySlugVersion("chirps-rain-season", "1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	if chirps.DataEnd == nil || chirps.DataEnd.Format("2006-01-02") != "2026-09-30" {
		t.Fatalf("CHIRPS data_end = %v, want 2026-09-30", chirps.DataEnd)
	}
	ndvi, err := repo.GetBySlugVersion("ndvi-s2-median", "1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	if ndvi.DataEnd != nil {
		t.Fatalf("NDVI has no coverage row, so data_end should be nil, got %v", ndvi.DataEnd)
	}

	methodologies, err := repo.List(newUser(t, tx, "auditor"))
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range methodologies {
		if (m.Slug == "chirps-rain-season") != (m.DataEnd != nil) {
			t.Errorf("List: %s data_end = %v", m.Slug, m.DataEnd)
		}
	}
}
