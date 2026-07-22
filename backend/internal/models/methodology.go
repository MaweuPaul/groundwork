package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

type Methodology struct {
	ID               uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Name             string         `gorm:"not null"`
	Version          string         `gorm:"not null"`
	Slug             string         `gorm:"not null"` // Unique combination of slug and version
	IndexType        string         `gorm:"not null"` // e.g. 'ndvi', 'evi'
	Satellite        string         `gorm:"not null"` // e.g. 'sentinel-2'
	Bands            pq.StringArray `gorm:"type:text[];not null"` // PostgreSQL array for text
	CloudMask        string         `gorm:"not null"` // e.g. 'qa60'
	MaxCloudPct      float64        `gorm:"default:20"`
	CompositeMethod  string         `gorm:"default:'median'"`
	DateLookbackDays int            `gorm:"default:30"`
	GeeScript        string         // Optional script text
	GeeScriptHash    string         // Optional script hash for verification
	CreatedAt        time.Time
	CreatedBy        string
}
