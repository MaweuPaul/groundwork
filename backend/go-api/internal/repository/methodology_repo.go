package repository

import (
	"errors"
	"groundwork/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MethodologyRepository interface {
	GetBySlugVersion(slug, version string) (*models.Methodology, error)
	List() ([]models.Methodology, error)
	Parameters(methodologyID uuid.UUID) ([]models.MethodologyParameter, error)
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

func (r *methodologyRepository) List() ([]models.Methodology, error) {
	methodologies := []models.Methodology{}
	if err := r.db.Order("slug, created_at").Find(&methodologies).Error; err != nil {
		return nil, err
	}

	var rows []struct {
		MethodologyID uuid.UUID
		models.MethodologyParameter
	}
	if err := r.db.Raw(`SELECT methodology_id, ` + parameterColumns + ` FROM methodology_parameters ORDER BY name`).
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	byMethodology := map[uuid.UUID][]models.MethodologyParameter{}
	for _, row := range rows {
		byMethodology[row.MethodologyID] = append(byMethodology[row.MethodologyID], row.MethodologyParameter)
	}
	for i := range methodologies {
		methodologies[i].Parameters = byMethodology[methodologies[i].ID]
		if methodologies[i].Parameters == nil {
			methodologies[i].Parameters = []models.MethodologyParameter{}
		}
	}
	return methodologies, nil
}

// The columns are NUMERIC, which would otherwise scan as text.
const parameterColumns = `name, label, type,
	min_value::float8 AS min_value, max_value::float8 AS max_value, default_value::float8 AS default_value`

// Parameters returns the methodology's parameters sorted by name; empty when it takes none.
func (r *methodologyRepository) Parameters(methodologyID uuid.UUID) ([]models.MethodologyParameter, error) {
	params := []models.MethodologyParameter{}
	err := r.db.Raw(`SELECT `+parameterColumns+` FROM methodology_parameters WHERE methodology_id = ? ORDER BY name`,
		methodologyID).Scan(&params).Error
	return params, err
}
