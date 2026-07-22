package models

import (
	"time"

	"github.com/google/uuid"
)

type ImagerySource struct {
	ID              uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	SnapshotID      uuid.UUID `gorm:"not null;index"`
	Satellite       string    `gorm:"not null"`
	ProductID       string    `gorm:"not null"`
	AcquisitionDate time.Time `gorm:"type:date;not null"`
	CloudCoverPct   float64
	TileID          string
	GeeAssetID      string
	CreatedAt       time.Time
}
