package repository

import (
	"errors"
	"groundwork/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

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
	err := r.db.Where("owner_id = ?", ownerID).
		Order("created_at DESC").
		Find(&parcels).Error
	return parcels, err
}

// GetByIDForOwner returns nil, nil when the parcel doesn't exist or isn't theirs.
func (r *parcelRepository) GetByIDForOwner(id, ownerID uuid.UUID) (*models.Parcel, error) {
	var parcel models.Parcel
	err := r.db.Where("id = ? AND owner_id = ?", id, ownerID).First(&parcel).Error
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
	return r.db.Create(parcel).Error
}
