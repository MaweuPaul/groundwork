package repository

import (
	"errors"
	"groundwork/internal/models"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MethodologyRepository interface {
	GetBySlugVersion(slug, version string) (*models.Methodology, error)
	List(ownerID uuid.UUID) ([]models.Methodology, error)
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
	var coverage []struct{ DataEnd time.Time }
	if err := r.db.Raw(`SELECT data_end FROM methodology_coverage WHERE methodology_id = ?`, m.ID).
		Scan(&coverage).Error; err != nil {
		return nil, err
	}
	if len(coverage) == 1 {
		m.DataEnd = &coverage[0].DataEnd
	}
	return &m, nil
}

// List returns every methodology with its parameters, measurements and manifest, and how
// many completed snapshots ownerID has made with each.
func (r *methodologyRepository) List(ownerID uuid.UUID) ([]models.Methodology, error) {
	methodologies := []models.Methodology{}
	if err := r.db.Order("created_at, slug").Find(&methodologies).Error; err != nil {
		return nil, err
	}

	var params []struct {
		MethodologyID uuid.UUID
		models.MethodologyParameter
	}
	if err := r.db.Raw(`SELECT methodology_id, ` + parameterColumns + ` FROM methodology_parameters ORDER BY name`).
		Scan(&params).Error; err != nil {
		return nil, err
	}

	var measurements []struct {
		MethodologyID uuid.UUID
		models.MethodologyMeasurement
	}
	if err := r.db.Raw(`
		SELECT mm.methodology_id, mt.slug, mt.name, mt.unit, COALESCE(mt.description, '') AS description
		FROM methodology_measurements mm
		JOIN measurement_types mt ON mt.id = mm.measurement_type_id
		ORDER BY mm.position`).Scan(&measurements).Error; err != nil {
		return nil, err
	}

	var extras []struct {
		ID             uuid.UUID
		ManifestText   string
		ManifestSHA256 string `gorm:"column:manifest_sha256"`
		Records        int
		DataEnd        *time.Time
	}
	if err := r.db.Raw(`
		SELECT m.id,
		       methodology_manifest_text(m.id) AS manifest_text,
		       methodology_manifest_hash(m.id) AS manifest_sha256,
		       (SELECT count(*)
		        FROM snapshots s JOIN parcels p ON p.id = s.parcel_id
		        WHERE s.methodology_id = m.id AND s.status = 'completed' AND p.owner_id = ?) AS records,
		       c.data_end
		FROM methodologies m
		LEFT JOIN methodology_coverage c ON c.methodology_id = m.id`, ownerID).Scan(&extras).Error; err != nil {
		return nil, err
	}

	index := make(map[uuid.UUID]*models.Methodology, len(methodologies))
	for i := range methodologies {
		m := &methodologies[i]
		m.Parameters, m.Measurements = []models.MethodologyParameter{}, []models.MethodologyMeasurement{}
		index[m.ID] = m
	}
	for _, p := range params {
		if m := index[p.MethodologyID]; m != nil {
			m.Parameters = append(m.Parameters, p.MethodologyParameter)
		}
	}
	for _, mm := range measurements {
		if m := index[mm.MethodologyID]; m != nil {
			m.Measurements = append(m.Measurements, mm.MethodologyMeasurement)
		}
	}
	for _, e := range extras {
		if m := index[e.ID]; m != nil {
			m.ManifestText, m.ManifestSHA256, m.Records, m.DataEnd = e.ManifestText, e.ManifestSHA256, e.Records, e.DataEnd
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
