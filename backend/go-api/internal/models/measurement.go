package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type Measurement struct {
	ID                uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	SnapshotID        uuid.UUID      `gorm:"type:uuid;not null;uniqueIndex:measurements_snapshot_type_unique" json:"snapshot_id"`
	MeasurementTypeID uuid.UUID      `gorm:"type:uuid;not null;uniqueIndex:measurements_snapshot_type_unique" json:"measurement_type_id"`
	Value             float64        `gorm:"type:double precision;not null" json:"value"`
	Metadata          datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'" json:"metadata"`
	CreatedAt         time.Time      `gorm:"type:timestamptz;not null;default:now()" json:"created_at"`

	Snapshot        Snapshot        `gorm:"foreignKey:SnapshotID" json:"snapshot"`
	MeasurementType MeasurementType `gorm:"foreignKey:MeasurementTypeID" json:"measurement_type"`
}
