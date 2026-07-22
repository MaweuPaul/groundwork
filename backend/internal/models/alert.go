package models

import (
	"time"

	"github.com/google/uuid"
)

type Alert struct {
	ID             uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	ParcelID       uuid.UUID `gorm:"not null;index"`
	SnapshotID     uuid.UUID `gorm:"not null;index"`
	AlertType      string    `gorm:"not null"` // e.g. 'ndvi_drop'
	Severity       string    `gorm:"not null"` // e.g. 'warning', 'critical'
	ThresholdValue float64   `gorm:"not null"`
	ActualValue    float64   `gorm:"not null"`
	PriorValue     float64
	Message        string
	Acknowledged   bool `gorm:"default:false"`
	CreatedAt      time.Time
}
