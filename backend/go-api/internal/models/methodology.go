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
	CloudMask        *string        `gorm:"type:text" json:"cloud_mask"`
	MaxCloudPct      float64        `gorm:"type:double precision;not null" json:"max_cloud_pct"`
	CompositeMethod  string         `gorm:"type:text;not null" json:"composite_method"`
	DateLookbackDays int            `gorm:"type:integer;not null" json:"date_lookback_days"`
	// First day the methodology's data covers; snapshots can't start earlier.
	DataStart time.Time `gorm:"type:date;not null" json:"data_start"`
	// Latest day the source has data for, as last checked by the worker; nil when unknown.
	// Snapshots can't end later. Filled by the repository.
	DataEnd   *time.Time `gorm:"-" json:"data_end"`
	CreatedAt time.Time  `gorm:"type:timestamptz;not null;default:now()" json:"created_at"`
	// The rest are filled by the repository's List.
	Parameters   []MethodologyParameter   `gorm:"-" json:"parameters"`
	Measurements []MethodologyMeasurement `gorm:"-" json:"measurements"`
	// The published spec every record made with this methodology seals by its hash.
	ManifestText   string `gorm:"-" json:"manifest_text"`
	ManifestSHA256 string `gorm:"-" json:"manifest_sha256"`
	// Completed snapshots the caller has made with it.
	Records int `gorm:"-" json:"records"`
}

// MethodologyMeasurement is one measurement a methodology produces, in display order.
type MethodologyMeasurement struct {
	Slug        string  `json:"slug"`
	Name        string  `json:"name"`
	Unit        *string `json:"unit"`
	Description string  `json:"description"`
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
