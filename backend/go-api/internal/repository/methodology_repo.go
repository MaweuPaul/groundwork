package repository

import (
	"errors"
	"groundwork/internal/models"

	"gorm.io/gorm"
)

type MethodologyRepository interface {
	GetBySlugVersion(slug, version string) (*models.Methodology, error)
}

type methodologyRepository struct {
	db *gorm.DB
}

func NewMethodologyRepository(db *gorm.DB) MethodologyRepository {
	return &methodologyRepository{db: db}
}

// GetBySlugVersion returns nil, nil when no such methodology version exists.
func (r *methodologyRepository) GetBySlugVersion(slug, version string) (*models.Methodology, error) {
	var m models.Methodology
	err := r.db.Where("slug = ? AND version = ?", slug, version).First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}
