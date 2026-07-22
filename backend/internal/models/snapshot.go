package models

import (
	"time"

	"github.com/google/uuid"
)

type SnapshotStatus string

const (
	StatusPending          SnapshotStatus = "pending"
	StatusRunning          SnapshotStatus = "running"
	StatusCompleted        SnapshotStatus = "completed"
	StatusFailed           SnapshotStatus = "failed"
	StatusInsufficientData SnapshotStatus = "insufficient_data"
)

type Snapshot struct {
	ID                 uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	ParcelID           uuid.UUID      `gorm:"not null;index"`
	MethodologyID      uuid.UUID      `gorm:"not null"`
	DateStart          time.Time      `gorm:"type:date;not null"`
	DateEnd            time.Time      `gorm:"type:date;not null"`
	Status             SnapshotStatus `gorm:"not null;default:'pending'"`
	ImageryCount       int
	TotalCloudPct      float64
	MethodologyVersion string         `gorm:"not null"` // Denormalized for audit
	GeometryHash       string         `gorm:"not null"` // Snapshot of boundary at computation time
	CreatedAt          time.Time
	CompletedAt        *time.Time
	Measurements       []Measurement   `gorm:"foreignKey:SnapshotID"`
	ImagerySources     []ImagerySource `gorm:"foreignKey:SnapshotID"`
}
