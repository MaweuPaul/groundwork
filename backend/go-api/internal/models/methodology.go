package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

type Methodology struct {
	ID               uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Name             string         `gorm:"type:text;not null" json:"name"`
	Slug             string         `gorm:"type:text;not null;unique" json:"slug"`
	Version          string         `gorm:"type:text;not null;uniqueIndex:methodologies_slug_version_key" json:"version"`
	Description      string         `gorm:"type:text" json:"description"`
	IndexType        string         `gorm:"type:text;not null" json:"index_type"`
	Satellite        string         `gorm:"type:text;not null" json:"satellite"`
	Bands            pq.StringArray `gorm:"type:text[];not null" json:"bands"`
	CloudMask        string         `gorm:"type:text;not null" json:"cloud_mask"`
	MaxCloudPct      float64        `gorm:"type:double precision;not null" json:"max_cloud_pct"`
	CompositeMethod  string         `gorm:"type:text;not null" json:"composite_method"`
	DateLookbackDays int            `gorm:"type:integer;not null" json:"date_lookback_days"`
	CreatedAt        time.Time      `gorm:"type:timestamptz;not null;default:now()" json:"created_at"`
	// From methodology_parameters; filled by the repository's List.
	Parameters []MethodologyParameter `gorm:"-" json:"parameters"`
}

// MethodologyParameter is one setting a methodology accepts, with its range and default.
type MethodologyParameter struct {
	Name    string  `json:"name"`
	Label   string  `json:"label"`
	Type    string  `json:"type"`
	Min     float64 `gorm:"column:min_value" json:"min"`
	Max     float64 `gorm:"column:max_value" json:"max"`
	Default float64 `gorm:"column:default_value" json:"default"`
}
