package repository

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// measurementsBySnapshot loads the measurements of many snapshots in one query,
// as type slug → value per snapshot. Snapshots without results get no entry.
func measurementsBySnapshot(db *gorm.DB, snapshotIDs []uuid.UUID) (map[uuid.UUID]map[string]float64, error) {
	out := make(map[uuid.UUID]map[string]float64, len(snapshotIDs))
	if len(snapshotIDs) == 0 {
		return out, nil
	}

	var rows []struct {
		SnapshotID uuid.UUID
		Slug       string
		Value      float64
	}
	err := db.Raw(`
		SELECT m.snapshot_id, mt.slug, m.value
		FROM measurements m
		JOIN measurement_types mt ON mt.id = m.measurement_type_id
		WHERE m.snapshot_id IN ?`,
		snapshotIDs,
	).Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	for _, row := range rows {
		if out[row.SnapshotID] == nil {
			out[row.SnapshotID] = map[string]float64{}
		}
		out[row.SnapshotID][row.Slug] = row.Value
	}
	return out, nil
}

// MeasurementType says how to label a measurement and what it means, as published by the
// methodology's migration.
type MeasurementType struct {
	Name        string  `json:"name"`
	Unit        *string `json:"unit"`
	Description *string `json:"description"`
}

// measurementTypesOf describes the measurements in values, by slug.
func measurementTypesOf(db *gorm.DB, values map[string]float64) (map[string]MeasurementType, error) {
	out := make(map[string]MeasurementType, len(values))
	if len(values) == 0 {
		return out, nil
	}
	slugs := make([]string, 0, len(values))
	for slug := range values {
		slugs = append(slugs, slug)
	}

	var rows []struct {
		Slug string
		MeasurementType
	}
	err := db.Raw(`SELECT slug, name, unit, description FROM measurement_types WHERE slug IN ?`, slugs).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.Slug] = row.MeasurementType
	}
	return out, nil
}
