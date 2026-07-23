package models

import (
	"time"

	"github.com/google/uuid"
)

type SnapshotStatus string

const (
	StatusPending    SnapshotStatus = "pending"
	StatusProcessing SnapshotStatus = "processing"
	StatusCompleted  SnapshotStatus = "completed"
	StatusFailed     SnapshotStatus = "failed"
)

type Snapshot struct {
	ID                 uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ParcelID           uuid.UUID      `gorm:"type:uuid;not null" json:"parcel_id"`
	MethodologyID      uuid.UUID      `gorm:"type:uuid;not null" json:"methodology_id"`
	DateStart          time.Time      `gorm:"type:date;not null" json:"date_start"`
	DateEnd            time.Time      `gorm:"type:date;not null" json:"date_end"`
	Status             SnapshotStatus `gorm:"type:snapshot_status;not null;default:'pending'" json:"status"`
	ParcelGeometry     string         `gorm:"type:geometry(Polygon,4326);not null" json:"parcel_geometry"`
	GeometryHash       string         `gorm:"type:text;not null" json:"geometry_hash"`
	MethodologyVersion string         `gorm:"type:text;not null" json:"methodology_version"`
	FailureMessage     *string        `gorm:"type:text" json:"failure_message,omitempty"`
	StartedAt          *time.Time     `gorm:"type:timestamptz" json:"started_at,omitempty"`
	CompletedAt        *time.Time     `gorm:"type:timestamptz" json:"completed_at,omitempty"`
	CreatedAt          time.Time      `gorm:"type:timestamptz;not null;default:now()" json:"created_at"`

	// Associations
	Parcel      Parcel      `gorm:"foreignKey:ParcelID" json:"parcel"`
	Methodology Methodology `gorm:"foreignKey:MethodologyID" json:"methodology"`
}
