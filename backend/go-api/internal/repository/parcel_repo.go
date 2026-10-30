package repository

import (
	"errors"
	"groundwork/internal/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

// ErrInvalidGeometry means PostGIS rejected the boundary (self-intersecting, overlapping pieces, bad GeoJSON).
var ErrInvalidGeometry = errors.New("invalid geometry")

// Columns to read back; geometry comes out as GeoJSON instead of hex.
const parcelColumns = `id, owner_id, name, description,
	ST_AsGeoJSON(geometry) AS geometry, area_ha, metadata, created_at, updated_at`

type ParcelRepository interface {
	Create(parcel *models.Parcel) error
	ListByOwner(ownerID uuid.UUID) ([]models.Parcel, error)
	GetByIDForOwner(id, ownerID uuid.UUID) (*models.Parcel, error)
}

type parcelRepository struct {
	db *gorm.DB
}

func (r *parcelRepository) ListByOwner(ownerID uuid.UUID) ([]models.Parcel, error) {
	var parcels []models.Parcel
	err := r.db.Select(parcelColumns).Where("owner_id = ?", ownerID).
		Order("created_at DESC").
		Find(&parcels).Error
	return parcels, err
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
	return &parcel, nil
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
