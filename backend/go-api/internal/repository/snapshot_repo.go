package repository

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ErrParcelNotFound means the parcel doesn't exist or isn't the caller's.
var ErrParcelNotFound = errors.New("parcel not found")

// NewSnapshot is what's needed to queue a snapshot. The geometry isn't here:
// it is copied from the parcel inside the database.
type NewSnapshot struct {
	ParcelID           uuid.UUID
	OwnerID            uuid.UUID
	MethodologyID      uuid.UUID
	MethodologyVersion string
	DateStart          time.Time
	DateEnd            time.Time
}

// SnapshotDetail is one snapshot as the API returns it: status, inputs, results
// and hash chain fields. The frozen geometry is left out; it's the heavy part and
// the client already has the parcel's boundary.
type SnapshotDetail struct {
	ID                 uuid.UUID          `json:"id"`
	ParcelID           uuid.UUID          `json:"parcel_id"`
	Status             string             `json:"status"`
	FailureMessage     *string            `json:"failure_message,omitempty"`
	Methodology        string             `json:"methodology"`
	MethodologyVersion string             `json:"methodology_version"`
	DateStart          string             `json:"date_start"` // YYYY-MM-DD
	DateEnd            string             `json:"date_end"`   // YYYY-MM-DD
	GeometryHash       string             `json:"geometry_hash"`
	CreatedAt          time.Time          `json:"created_at"`
	StartedAt          *time.Time         `json:"started_at,omitempty"`
	CompletedAt        *time.Time         `json:"completed_at,omitempty"`
	ChainSeq           *int64             `json:"chain_seq,omitempty"`
	RecordHash         *string            `json:"record_hash,omitempty"`
	PrevRecordHash     *string            `json:"prev_record_hash,omitempty"`
	Job                *SnapshotJobInfo   `json:"job,omitempty"`
	Measurements       map[string]float64 `json:"measurements"` // type slug → value
	ImagerySources     []ImagerySourceRef `json:"imagery_sources"`
}

// SnapshotJobInfo is the queue progress shown to the client. The worker's raw
// error message stays internal; only its code is shown.
type SnapshotJobInfo struct {
	Status        string     `json:"status"`
	Attempts      int        `json:"attempts"`
	MaxAttempts   int        `json:"max_attempts"`
	NextAttemptAt *time.Time `json:"next_attempt_at,omitempty"` // set while waiting to retry
	LastErrorCode *string    `json:"last_error_code,omitempty"`
}

type ImagerySourceRef struct {
	SceneID         string    `json:"scene_id"`
	Satellite       string    `json:"satellite"`
	AcquisitionTime time.Time `json:"acquisition_time"`
	CloudCoverPct   *float64  `json:"cloud_cover_pct,omitempty"`
}

type SnapshotRepository interface {
	CreateWithJob(in NewSnapshot) (uuid.UUID, error)
	GetByIDForOwner(id, ownerID uuid.UUID) (*SnapshotDetail, error)
}

type snapshotRepository struct {
	db *gorm.DB
}

func NewSnapshotRepository(db *gorm.DB) SnapshotRepository {
	return &snapshotRepository{db: db}
}

// CreateWithJob inserts a pending snapshot and its queued job in one transaction,
// so a snapshot never exists without a job. The insert selects from the parcel,
// which checks ownership and freezes the current geometry in the same statement.
//
// geometry_hash is taken from a normalised copy (coordinates snapped to ~1 cm,
// points in a standard order), so the same boundary always hashes the same,
// however it was drawn.
func (r *snapshotRepository) CreateWithJob(in NewSnapshot) (uuid.UUID, error) {
	var row struct{ ID uuid.UUID }

	err := r.db.Transaction(func(tx *gorm.DB) error {
		res := tx.Raw(`
			INSERT INTO snapshots (parcel_id, methodology_id, methodology_version,
			                       date_start, date_end, parcel_geometry, geometry_hash)
			SELECT p.id, ?, ?, ?::date, ?::date,
			       p.geometry,
			       encode(sha256(ST_AsEWKB(ST_Normalize(ST_SnapToGrid(p.geometry, 0.0000001)), 'NDR')), 'hex')
			FROM parcels p
			WHERE p.id = ? AND p.owner_id = ?
			RETURNING id`,
			in.MethodologyID, in.MethodologyVersion,
			in.DateStart.Format(time.DateOnly), in.DateEnd.Format(time.DateOnly),
			in.ParcelID, in.OwnerID,
		).Scan(&row)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrParcelNotFound
		}

		// The insert trigger notifies the worker once this transaction commits.
		return tx.Exec(`INSERT INTO snapshot_jobs (snapshot_id) VALUES (?)`, row.ID).Error
	})
	if err != nil {
		return uuid.Nil, err
	}
	return row.ID, nil
}

// GetByIDForOwner returns nil, nil when the snapshot doesn't exist or its parcel
// isn't the owner's. Snapshots have no owner column; ownership comes from the parcel.
func (r *snapshotRepository) GetByIDForOwner(id, ownerID uuid.UUID) (*SnapshotDetail, error) {
	var row struct {
		ID                 uuid.UUID
		ParcelID           uuid.UUID
		Status             string
		FailureMessage     *string
		Methodology        string
		MethodologyVersion string
		DateStart          string
		DateEnd            string
		GeometryHash       string
		CreatedAt          time.Time
		StartedAt          *time.Time
		CompletedAt        *time.Time
		ChainSeq           *int64
		RecordHash         *string
		PrevRecordHash     *string
		JobStatus          *string
		JobAttempts        *int
		JobMaxAttempts     *int
		JobRunAfter        *time.Time
		JobLastErrorCode   *string
	}

	res := r.db.Raw(`
		SELECT s.id, s.parcel_id, s.status, s.failure_message,
		       m.slug AS methodology, s.methodology_version,
		       to_char(s.date_start, 'YYYY-MM-DD') AS date_start,
		       to_char(s.date_end, 'YYYY-MM-DD') AS date_end,
		       s.geometry_hash, s.created_at, s.started_at, s.completed_at,
		       s.chain_seq, s.record_hash, s.prev_record_hash,
		       j.status AS job_status, j.attempts AS job_attempts,
		       j.max_attempts AS job_max_attempts, j.run_after AS job_run_after,
		       j.last_error_code AS job_last_error_code
		FROM snapshots s
		JOIN parcels p ON p.id = s.parcel_id
		JOIN methodologies m ON m.id = s.methodology_id
		LEFT JOIN snapshot_jobs j ON j.snapshot_id = s.id
		WHERE s.id = ? AND p.owner_id = ?`,
		id, ownerID,
	).Scan(&row)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, nil
	}

	detail := &SnapshotDetail{
		ID:                 row.ID,
		ParcelID:           row.ParcelID,
		Status:             row.Status,
		FailureMessage:     row.FailureMessage,
		Methodology:        row.Methodology,
		MethodologyVersion: row.MethodologyVersion,
		DateStart:          row.DateStart,
		DateEnd:            row.DateEnd,
		GeometryHash:       row.GeometryHash,
		CreatedAt:          row.CreatedAt,
		StartedAt:          row.StartedAt,
		CompletedAt:        row.CompletedAt,
		ChainSeq:           row.ChainSeq,
		RecordHash:         row.RecordHash,
		PrevRecordHash:     row.PrevRecordHash,
		Measurements:       map[string]float64{},
		ImagerySources:     []ImagerySourceRef{},
	}

	if row.JobStatus != nil {
		detail.Job = &SnapshotJobInfo{
			Status:        *row.JobStatus,
			Attempts:      *row.JobAttempts,
			MaxAttempts:   *row.JobMaxAttempts,
			LastErrorCode: row.JobLastErrorCode,
		}
		// run_after only means something while a failed attempt waits to retry.
		if *row.JobStatus == "queued" && *row.JobAttempts > 0 {
			detail.Job.NextAttemptAt = row.JobRunAfter
		}
	}

	var measurements []struct {
		Slug  string
		Value float64
	}
	err := r.db.Raw(`
		SELECT mt.slug, m.value
		FROM measurements m
		JOIN measurement_types mt ON mt.id = m.measurement_type_id
		WHERE m.snapshot_id = ?`,
		id,
	).Scan(&measurements).Error
	if err != nil {
		return nil, err
	}
	for _, m := range measurements {
		detail.Measurements[m.Slug] = m.Value
	}

	err = r.db.Raw(`
		SELECT scene_id, satellite, acquisition_time, cloud_cover_pct
		FROM imagery_sources
		WHERE snapshot_id = ?
		ORDER BY acquisition_time, scene_id`,
		id,
	).Scan(&detail.ImagerySources).Error
	if err != nil {
		return nil, err
	}

	return detail, nil
}
