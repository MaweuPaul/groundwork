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
	// Sealed into the record: scene-level cloud prefilter and Cloud Score+ clear threshold.
	MaxSceneCloudPct    float64
	CloudScoreThreshold float64
}

// SnapshotDetail is one snapshot as the API returns it: status, inputs, results
// and hash chain fields. The frozen geometry is left out; it's the heavy part and
// the client already has the parcel's boundary.
type SnapshotDetail struct {
	ID                  uuid.UUID          `json:"id"`
	ShortID             string             `json:"short_id"`
	ParcelID            uuid.UUID          `json:"parcel_id"`
	ParcelShortID       string             `json:"parcel_short_id"`
	ParcelName          string             `json:"parcel_name"`
	Status              string             `json:"status"`
	FailureMessage      *string            `json:"failure_message,omitempty"`
	Methodology         string             `json:"methodology"`
	MethodologyVersion  string             `json:"methodology_version"`
	DateStart           string             `json:"date_start"` // YYYY-MM-DD
	DateEnd             string             `json:"date_end"`   // YYYY-MM-DD
	MaxSceneCloudPct    float64            `json:"max_scene_cloud_pct"`
	CloudScoreThreshold float64            `json:"cloud_score_threshold"`
	RecordFormat        int                `json:"record_format"`
	GeometryHash        string             `json:"geometry_hash"`
	CreatedAt           time.Time          `json:"created_at"`
	StartedAt           *time.Time         `json:"started_at,omitempty"`
	CompletedAt         *time.Time         `json:"completed_at,omitempty"`
	ChainSeq            *int64             `json:"chain_seq,omitempty"`
	RecordHash          *string            `json:"record_hash,omitempty"`
	PrevRecordHash      *string            `json:"prev_record_hash,omitempty"`
	Job                 *SnapshotJobInfo   `json:"job,omitempty"`
	Measurements        map[string]float64 `json:"measurements"` // type slug → value
	ImagerySources      []ImagerySourceRef `json:"imagery_sources"`
	NdviImage           *NdviImageInfo     `json:"ndvi_image,omitempty"`
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
	ListByParcelForOwner(parcelID, ownerID uuid.UUID) ([]SnapshotSummary, error)
	NdviPNGForOwner(id, ownerID uuid.UUID) ([]byte, error)
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
			                       date_start, date_end, max_scene_cloud_pct, cloud_score_threshold,
			                       parcel_geometry, geometry_hash)
			SELECT p.id, ?, ?, ?::date, ?::date, ?, ?,
			       p.geometry,
			       encode(sha256(ST_AsEWKB(ST_Normalize(ST_SnapToGrid(p.geometry, 0.0000001)), 'NDR')), 'hex')
			FROM parcels p
			WHERE p.id = ? AND p.owner_id = ?
			RETURNING id`,
			in.MethodologyID, in.MethodologyVersion,
			in.DateStart.Format(time.DateOnly), in.DateEnd.Format(time.DateOnly),
			in.MaxSceneCloudPct, in.CloudScoreThreshold,
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
		ID                  uuid.UUID
		ShortID             string
		ParcelID            uuid.UUID
		ParcelShortID       string
		ParcelName          string
		Status              string
		FailureMessage      *string
		Methodology         string
		MethodologyVersion  string
		DateStart           string
		DateEnd             string
		MaxSceneCloudPct    float64
		CloudScoreThreshold float64
		RecordFormat        int
		GeometryHash        string
		CreatedAt           time.Time
		StartedAt           *time.Time
		CompletedAt         *time.Time
		ChainSeq            *int64
		RecordHash          *string
		PrevRecordHash      *string
		JobStatus           *string
		JobAttempts         *int
		JobMaxAttempts      *int
		JobRunAfter         *time.Time
		JobLastErrorCode    *string
	}

	res := r.db.Raw(`
		SELECT s.id, s.short_id, s.parcel_id, p.short_id AS parcel_short_id,
		       p.name AS parcel_name, s.status, s.failure_message,
		       m.slug AS methodology, s.methodology_version,
		       to_char(s.date_start, 'YYYY-MM-DD') AS date_start,
		       to_char(s.date_end, 'YYYY-MM-DD') AS date_end,
		       s.max_scene_cloud_pct, s.cloud_score_threshold, s.record_format,
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
		ID:                  row.ID,
		ShortID:             row.ShortID,
		ParcelID:            row.ParcelID,
		ParcelShortID:       row.ParcelShortID,
		ParcelName:          row.ParcelName,
		Status:              row.Status,
		FailureMessage:      row.FailureMessage,
		Methodology:         row.Methodology,
		MethodologyVersion:  row.MethodologyVersion,
		DateStart:           row.DateStart,
		DateEnd:             row.DateEnd,
		MaxSceneCloudPct:    row.MaxSceneCloudPct,
		CloudScoreThreshold: row.CloudScoreThreshold,
		RecordFormat:        row.RecordFormat,
		GeometryHash:        row.GeometryHash,
		CreatedAt:           row.CreatedAt,
		StartedAt:           row.StartedAt,
		CompletedAt:         row.CompletedAt,
		ChainSeq:            row.ChainSeq,
		RecordHash:          row.RecordHash,
		PrevRecordHash:      row.PrevRecordHash,
		Measurements:        map[string]float64{},
		ImagerySources:      []ImagerySourceRef{},
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

	measurements, err := measurementsBySnapshot(r.db, []uuid.UUID{id})
	if err != nil {
		return nil, err
	}
	if m := measurements[id]; m != nil {
		detail.Measurements = m
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

	images, err := ndviImagesBySnapshot(r.db, []uuid.UUID{id})
	if err != nil {
		return nil, err
	}
	detail.NdviImage = images[id]

	return detail, nil
}

// SnapshotSummary is one row of a parcel's snapshot history.
type SnapshotSummary struct {
	ID                  uuid.UUID          `json:"id"`
	ShortID             string             `json:"short_id"`
	Status              string             `json:"status"`
	FailureMessage      *string            `json:"failure_message,omitempty"`
	Methodology         string             `json:"methodology"`
	MethodologyVersion  string             `json:"methodology_version"`
	DateStart           string             `json:"date_start"`
	DateEnd             string             `json:"date_end"`
	MaxSceneCloudPct    float64            `json:"max_scene_cloud_pct"`
	CloudScoreThreshold float64            `json:"cloud_score_threshold"`
	CreatedAt           time.Time          `json:"created_at"`
	CompletedAt         *time.Time         `json:"completed_at,omitempty"`
	ChainSeq            *int64             `json:"chain_seq,omitempty"`
	RecordHash          *string            `json:"record_hash,omitempty"`
	Measurements        map[string]float64 `json:"measurements"`
	NdviImage           *NdviImageInfo     `json:"ndvi_image,omitempty"`
}

// ListByParcelForOwner returns the parcel's snapshots, newest first, or
// ErrParcelNotFound when the parcel doesn't exist or isn't the owner's.
func (r *snapshotRepository) ListByParcelForOwner(parcelID, ownerID uuid.UUID) ([]SnapshotSummary, error) {
	var owned bool
	err := r.db.Raw(`SELECT EXISTS (SELECT 1 FROM parcels WHERE id = ? AND owner_id = ?)`, parcelID, ownerID).
		Scan(&owned).Error
	if err != nil {
		return nil, err
	}
	if !owned {
		return nil, ErrParcelNotFound
	}

	snapshots := []SnapshotSummary{}
	err = r.db.Raw(`
		SELECT s.id, s.short_id, s.status, s.failure_message, m.slug AS methodology, s.methodology_version,
		       to_char(s.date_start, 'YYYY-MM-DD') AS date_start,
		       to_char(s.date_end, 'YYYY-MM-DD') AS date_end,
		       s.max_scene_cloud_pct, s.cloud_score_threshold,
		       s.created_at, s.completed_at, s.chain_seq, s.record_hash
		FROM snapshots s
		JOIN methodologies m ON m.id = s.methodology_id
		WHERE s.parcel_id = ?
		ORDER BY s.created_at DESC`,
		parcelID,
	).Scan(&snapshots).Error
	if err != nil {
		return nil, err
	}

	ids := make([]uuid.UUID, len(snapshots))
	for i, s := range snapshots {
		ids[i] = s.ID
	}
	measurements, err := measurementsBySnapshot(r.db, ids)
	if err != nil {
		return nil, err
	}
	images, err := ndviImagesBySnapshot(r.db, ids)
	if err != nil {
		return nil, err
	}
	for i := range snapshots {
		snapshots[i].Measurements = measurements[snapshots[i].ID]
		if snapshots[i].Measurements == nil {
			snapshots[i].Measurements = map[string]float64{}
		}
		snapshots[i].NdviImage = images[snapshots[i].ID]
	}

	return snapshots, nil
}
