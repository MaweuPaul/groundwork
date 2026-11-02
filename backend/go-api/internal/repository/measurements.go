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
