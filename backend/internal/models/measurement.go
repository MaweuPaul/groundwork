package models

import (
	"time"

	"github.com/google/uuid"
)

type Measurement struct {
	ID           uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	SnapshotID   uuid.UUID `gorm:"not null;index"`
	IndexType    string    `gorm:"not null"`
	ValueMean    float64
	ValueMin     float64
	ValueMax     float64
	ValueStddev  float64
	PixelCount   int
	TrendVsPrior float64
	PctChange    float64
	CreatedAt    time.Time
}
