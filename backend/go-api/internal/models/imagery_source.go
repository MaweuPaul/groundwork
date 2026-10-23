package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type ImagerySource struct {
	ID              uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	SnapshotID      uuid.UUID      `gorm:"type:uuid;not null;uniqueIndex:imagery_sources_snapshot_scene_unique" json:"snapshot_id"`
	Provider        string         `gorm:"type:text;not null" json:"provider"`
	Satellite       string         `gorm:"type:text;not null" json:"satellite"`
	SceneID         string         `gorm:"type:text;not null;uniqueIndex:imagery_sources_snapshot_scene_unique" json:"scene_id"`
	AcquisitionTime time.Time      `gorm:"type:timestamptz;not null" json:"acquisition_time"`
	CloudCoverPct   *float64       `gorm:"type:double precision" json:"cloud_cover_pct,omitempty"`
	ProcessingLevel string         `gorm:"type:text;not null" json:"processing_level"`
	AssetURI        *string        `gorm:"type:text" json:"asset_uri,omitempty"`
	Metadata        datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'" json:"metadata"`
	CreatedAt       time.Time      `gorm:"type:timestamptz;not null;default:now()" json:"created_at"`

	Snapshot Snapshot `gorm:"foreignKey:SnapshotID" json:"snapshot"`
}
