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

type SnapshotRepository interface {
	CreateWithJob(in NewSnapshot) (uuid.UUID, error)
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
