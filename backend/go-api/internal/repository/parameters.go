package repository

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Parameters is a snapshot's methodology settings, stored as a JSONB object of numbers.
type Parameters map[string]float64

func (p *Parameters) Scan(src any) error {
	var raw []byte
	switch v := src.(type) {
	case []byte:
		raw = v
	case string:
		raw = []byte(v)
	case nil:
		*p = Parameters{}
		return nil
	default:
		return fmt.Errorf("parameters: cannot scan %T", src)
	}
	return json.Unmarshal(raw, p)
}

func (p Parameters) Value() (driver.Value, error) {
	if p == nil {
		return "{}", nil
	}
	b, err := json.Marshal(p)
	return string(b), err
}

// SnapshotFlag is a quality flag the worker raised, e.g. low_coverage. It doesn't fail the snapshot.
type SnapshotFlag struct {
	Flag   string  `json:"flag"`
	Detail *string `json:"detail,omitempty"`
}

// flagsBySnapshot loads the flags of many snapshots in one query.
func flagsBySnapshot(db *gorm.DB, snapshotIDs []uuid.UUID) (map[uuid.UUID][]SnapshotFlag, error) {
	out := make(map[uuid.UUID][]SnapshotFlag, len(snapshotIDs))
	if len(snapshotIDs) == 0 {
		return out, nil
	}

	var rows []struct {
		SnapshotID uuid.UUID
		SnapshotFlag
	}
	err := db.Raw(`SELECT snapshot_id, flag, detail FROM snapshot_flags WHERE snapshot_id IN ? ORDER BY flag`, snapshotIDs).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.SnapshotID] = append(out[row.SnapshotID], row.SnapshotFlag)
	}
	return out, nil
}

func flagsOrEmpty(flags []SnapshotFlag) []SnapshotFlag {
	if flags == nil {
		return []SnapshotFlag{}
	}
	return flags
}
