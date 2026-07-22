package repository

import (
	"groundwork/internal/models"

	"gorm.io/gorm"
)

type ParcelRepository interface {
	Create(parcel *models.Parcel) error
	GetAll() ([]models.Parcel, error)
}

type parcelRepository struct {
	db *gorm.DB
}

func NewParcelRepository(db *gorm.DB) ParcelRepository {
	return &parcelRepository{db: db}
}

func (r *parcelRepository) Create(parcel *models.Parcel) error {
	return r.db.Create(parcel).Error
}

func (r *parcelRepository) GetAll() ([]models.Parcel, error) {
	var parcels []models.Parcel
	err := r.db.Find(&parcels).Error
	return parcels, err
}
