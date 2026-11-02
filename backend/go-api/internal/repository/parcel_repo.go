package repository

import (
	"errors"
	"groundwork/internal/models"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

// ErrInvalidGeometry means PostGIS rejected the boundary (self-intersecting, overlapping pieces, bad GeoJSON).
var ErrInvalidGeometry = errors.New("invalid geometry")

// Columns to read back; geometry comes out as GeoJSON instead of hex.
const parcelColumns = `id, owner_id, name, description,
	ST_AsGeoJSON(geometry) AS geometry, area_ha, metadata, created_at, updated_at,
	ST_X(ST_Centroid(geometry)) AS centroid_lon, ST_Y(ST_Centroid(geometry)) AS centroid_lat`

type ParcelRepository interface {
	Create(parcel *models.Parcel) error
	ListByOwner(ownerID uuid.UUID) ([]models.Parcel, error)
	GetByIDForOwner(id, ownerID uuid.UUID) (*models.Parcel, error)
}

type parcelRepository struct {
	db *gorm.DB
}

func (r *parcelRepository) ListByOwner(ownerID uuid.UUID) ([]models.Parcel, error) {
	parcels := []models.Parcel{}
	err := r.db.Select(parcelColumns).Where("owner_id = ?", ownerID).
		Order("created_at DESC").
		Find(&parcels).Error
	if err != nil {
		return nil, err
	}
	return parcels, r.attachLatestSnapshots(parcels)
}

// GetByIDForOwner returns nil, nil when the parcel doesn't exist or isn't theirs.
func (r *parcelRepository) GetByIDForOwner(id, ownerID uuid.UUID) (*models.Parcel, error) {
	var parcel models.Parcel
	err := r.db.Select(parcelColumns).Where("id = ? AND owner_id = ?", id, ownerID).First(&parcel).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	parcels := []models.Parcel{parcel}
	if err := r.attachLatestSnapshots(parcels); err != nil {
		return nil, err
	}
	return &parcels[0], nil
}

// attachLatestSnapshots fills each parcel's latest snapshot (any status) and the
// results of its latest completed one, with a fixed number of queries.
func (r *parcelRepository) attachLatestSnapshots(parcels []models.Parcel) error {
	if len(parcels) == 0 {
		return nil
	}
	ids := make([]uuid.UUID, len(parcels))
	for i, p := range parcels {
		ids[i] = p.ID
	}

	var latest []struct {
		ParcelID uuid.UUID
		models.SnapshotRef
	}
	err := r.db.Raw(`
		SELECT DISTINCT ON (parcel_id) parcel_id, id, status, created_at, completed_at
		FROM snapshots
		WHERE parcel_id IN ?
		ORDER BY parcel_id, created_at DESC`,
		ids,
	).Scan(&latest).Error
	if err != nil {
		return err
	}

	var completed []struct {
		ParcelID    uuid.UUID
		SnapshotID  uuid.UUID
		CompletedAt time.Time
	}
	err = r.db.Raw(`
		SELECT DISTINCT ON (parcel_id) parcel_id, id AS snapshot_id, completed_at
		FROM snapshots
		WHERE parcel_id IN ? AND status = 'completed'
		ORDER BY parcel_id, completed_at DESC`,
		ids,
	).Scan(&completed).Error
	if err != nil {
		return err
	}

	completedIDs := make([]uuid.UUID, len(completed))
	for i, c := range completed {
		completedIDs[i] = c.SnapshotID
	}
	measurements, err := measurementsBySnapshot(r.db, completedIDs)
	if err != nil {
		return err
	}

	byParcel := make(map[uuid.UUID]int, len(parcels))
	for i, p := range parcels {
		byParcel[p.ID] = i
	}
	for _, l := range latest {
		ref := l.SnapshotRef
		parcels[byParcel[l.ParcelID]].LatestSnapshot = &ref
	}
	for _, c := range completed {
		parcels[byParcel[c.ParcelID]].LatestResult = &models.SnapshotResult{
			SnapshotID:   c.SnapshotID,
			CompletedAt:  c.CompletedAt,
			Measurements: measurements[c.SnapshotID],
		}
	}
	return nil
}

func NewParcelRepository(db *gorm.DB) ParcelRepository {
	return &parcelRepository{db: db}
}

func (r *parcelRepository) Create(parcel *models.Parcel) error {
	err := r.db.Raw(`
		INSERT INTO parcels (owner_id, name, description, geometry, metadata)
		VALUES (?, ?, ?, ST_Multi(ST_GeomFromGeoJSON(?)), ?)
		RETURNING `+parcelColumns,
		parcel.OwnerID, parcel.Name, parcel.Description, string(parcel.Geometry), parcel.Metadata,
	).Scan(parcel).Error

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && (pgErr.Code == "23514" || pgErr.Code == "22023" || pgErr.Code == "XX000") {
		return ErrInvalidGeometry
	}
	return err
}
