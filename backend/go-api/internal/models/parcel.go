package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type Parcel struct {
	ID          uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ShortID     string         `gorm:"->" json:"short_id"` // Read-only, assigned by the database; used in URLs
	OwnerID     uuid.UUID      `gorm:"type:uuid;not null" json:"owner_id"`
	Name        string         `gorm:"type:text;not null" json:"name"`
	Description string         `gorm:"type:text" json:"description"`
	Geometry    datatypes.JSON `gorm:"type:geometry(MultiPolygon,4326);not null" json:"geometry"`
	AreaHa      float64        `gorm:"->;type:double precision" json:"area_ha"` // Read-only, computed by PostGIS
	Metadata    datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'" json:"metadata"`
	CreatedAt   time.Time      `gorm:"type:timestamptz;not null;default:now()" json:"created_at"`
	UpdatedAt   time.Time      `gorm:"type:timestamptz;not null;default:now()" json:"updated_at"`

	CentroidLon float64 `gorm:"->" json:"centroid_lon"`
	CentroidLat float64 `gorm:"->" json:"centroid_lat"`

	LatestSnapshot *SnapshotRef    `gorm:"-" json:"latest_snapshot"`
	LatestResult   *SnapshotResult `gorm:"-" json:"latest_result"`
}

// SnapshotRef is a parcel's most recent snapshot, whatever its status.
type SnapshotRef struct {
	ID          uuid.UUID  `json:"id"`
	ShortID     string     `json:"short_id"`
	Status      string     `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

// SnapshotResult is a parcel's most recent completed snapshot.
type SnapshotResult struct {
	SnapshotID   uuid.UUID          `json:"snapshot_id"`
	CompletedAt  time.Time          `json:"completed_at"`
	Measurements map[string]float64 `json:"measurements"`
}
