package integration

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"groundwork/internal/models"
	"groundwork/internal/repository"
)

const worker = "integration-worker"

// A small square in Nyeri, Kenya.
const squareGeoJSON = `{"type":"Polygon","coordinates":[[[36.95,-0.42],[36.96,-0.42],[36.96,-0.41],[36.95,-0.41],[36.95,-0.42]]]}`

func newUser(t *testing.T, tx *gorm.DB, role string) uuid.UUID {
	t.Helper()
	var row struct{ ID uuid.UUID }
	name := "user-" + uuid.NewString()
	err := tx.Raw(`
		INSERT INTO users (email, username, password, first_name, last_name, role)
		VALUES (?, ?, 'not-a-real-hash', 'Test', 'User', ?)
		RETURNING id`,
		name+"@example.test", name, role,
	).Scan(&row).Error
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}
	return row.ID
}

func newParcel(t *testing.T, tx *gorm.DB, owner uuid.UUID, name string) models.Parcel {
	t.Helper()
	parcel := models.Parcel{
		OwnerID:  owner,
		Name:     name,
		Geometry: datatypes.JSON(squareGeoJSON),
		Metadata: datatypes.JSON(`{}`),
	}
	if err := repository.NewParcelRepository(tx).Create(&parcel); err != nil {
		t.Fatalf("create parcel: %v", err)
	}
	return parcel
}

type methodologyRef struct {
	ID      uuid.UUID
	Version string
}

func ndvi(t *testing.T, tx *gorm.DB) methodologyRef {
	t.Helper()
	return methodology(t, tx, "ndvi-s2-median")
}

func methodology(t *testing.T, tx *gorm.DB, slug string) methodologyRef {
	t.Helper()
	var m methodologyRef
	err := tx.Raw(`SELECT id, version FROM methodologies WHERE slug = ? AND version = '1.0.0'`, slug).Scan(&m).Error
	if err != nil || m.ID == uuid.Nil {
		t.Fatalf("%s methodology not seeded: %v", slug, err)
	}
	return m
}

func day(s string) time.Time {
	d, err := time.Parse(time.DateOnly, s)
	if err != nil {
		panic(err)
	}
	return d
}

func queueSnapshot(t *testing.T, tx *gorm.DB, owner uuid.UUID, parcel models.Parcel, params repository.Parameters) uuid.UUID {
	t.Helper()
	m := ndvi(t, tx)
	id, err := repository.NewSnapshotRepository(tx).CreateWithJob(repository.NewSnapshot{
		ParcelID:           parcel.ID,
		OwnerID:            owner,
		MethodologyID:      m.ID,
		MethodologyVersion: m.Version,
		DateStart:          day("2026-10-01"),
		DateEnd:            day("2026-10-30"),
		Parameters:         params,
	})
	if err != nil {
		t.Fatalf("queue snapshot: %v", err)
	}
	return id
}

type claimedJob struct {
	JobID      uuid.UUID
	SnapshotID uuid.UUID
	Attempt    int
	Parameters repository.Parameters
}

// claim takes the next due job as the worker does, or returns nil when none is due.
func claim(t *testing.T, tx *gorm.DB, as string) *claimedJob {
	t.Helper()
	var jobs []claimedJob
	err := tx.Raw(`SELECT job_id, snapshot_id, attempt, parameters FROM claim_snapshot_job(?, interval '10 minutes')`, as).
		Scan(&jobs).Error
	if err != nil {
		t.Fatalf("claim: %v", err)
	}
	if len(jobs) == 0 || jobs[0].JobID == uuid.Nil {
		return nil
	}
	return &jobs[0]
}

// currentFormat is the record format new snapshots are sealed with.
const currentFormat = 5

type result struct {
	Measurements map[string]float64
	Sources      []string // asset URIs, all at version "v1"
	Flags        map[string]string
	Artifacts    []artifact
}

type artifact struct {
	Key, MediaType, Metadata string
	Content                  []byte
}

var clearResult = result{
	Measurements: map[string]float64{"ndvi_mean": 0.612345678, "ndvi_max": 0.9, "valid_pixel_pct": 87.5},
	Sources:      []string{"COPERNICUS/S2_SR_HARMONIZED/20261005T074619_20261005T080046_T37MBU"},
}

// save writes results the way the worker's save_and_complete does, without completing.
func save(t *testing.T, tx *gorm.DB, snapshotID uuid.UUID, r result) {
	t.Helper()
	for slug, value := range r.Measurements {
		err := tx.Exec(`
			INSERT INTO measurements (snapshot_id, measurement_type_id, value, metadata)
			SELECT ?, id, ?, '{}'::jsonb FROM measurement_types WHERE slug = ?`,
			snapshotID, value, slug,
		).Error
		if err != nil {
			t.Fatalf("insert measurement %s: %v", slug, err)
		}
	}
	for i, uri := range r.Sources {
		err := tx.Exec(`
			INSERT INTO imagery_sources (snapshot_id, provider, satellite, scene_id, acquisition_time,
			                             cloud_cover_pct, processing_level, asset_uri, source_version)
			VALUES (?, 'earth-engine', 'sentinel-2', ?, '2026-10-05T07:46:19Z', 12.5, 'L2A', ?, 'v1')`,
			snapshotID, uri[strings.LastIndex(uri, "/")+1:]+strings.Repeat("x", i), uri,
		).Error
		if err != nil {
			t.Fatalf("insert source: %v", err)
		}
	}
	for flag, detail := range r.Flags {
		if err := tx.Exec(`INSERT INTO snapshot_flags (snapshot_id, flag, detail) VALUES (?, ?, ?)`,
			snapshotID, flag, detail).Error; err != nil {
			t.Fatalf("insert flag %s: %v", flag, err)
		}
	}
	for _, a := range r.Artifacts {
		if err := tx.Exec(`INSERT INTO artifacts (snapshot_id, key, media_type, content, metadata) VALUES (?, ?, ?, ?, ?::jsonb)`,
			snapshotID, a.Key, a.MediaType, a.Content, a.Metadata).Error; err != nil {
			t.Fatalf("insert artifact %s: %v", a.Key, err)
		}
	}
}

// codeVersion is what the test worker reports as the code that computed its results.
const codeVersion = "0123456789abcdef0123456789abcdef01234567"

// complete saves the results and completes the job, returning the record hash.
func complete(t *testing.T, tx *gorm.DB, job *claimedJob, r result) string {
	t.Helper()
	save(t, tx, job.SnapshotID, r)
	var hash string
	if err := tx.Raw(`SELECT complete_snapshot_job(?, ?, ?)`, job.JobID, worker, codeVersion).Scan(&hash).Error; err != nil {
		t.Fatalf("complete: %v", err)
	}
	return hash
}

// runSnapshot queues, claims and completes one snapshot.
func runSnapshot(t *testing.T, tx *gorm.DB, owner uuid.UUID, parcel models.Parcel, r result) (uuid.UUID, string) {
	t.Helper()
	id := queueSnapshot(t, tx, owner, parcel, nil)
	job := claim(t, tx, worker)
	if job == nil || job.SnapshotID != id {
		t.Fatalf("expected to claim snapshot %s, got %+v", id, job)
	}
	return id, complete(t, tx, job, r)
}

func chainProblems(t *testing.T, tx *gorm.DB) []string {
	t.Helper()
	var problems []string
	if err := tx.Raw(`SELECT problem FROM verify_snapshot_chain()`).Scan(&problems).Error; err != nil {
		t.Fatalf("verify chain: %v", err)
	}
	return problems
}

// expectError runs fn in a savepoint, so the failure doesn't abort the test's transaction,
// and checks that it fails with a message containing want.
func expectError(t *testing.T, tx *gorm.DB, want string, fn func(tx *gorm.DB) error) {
	t.Helper()
	err := tx.Transaction(fn)
	if err == nil {
		t.Fatalf("expected an error containing %q, got none", want)
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("expected an error containing %q, got: %v", want, err)
	}
}
