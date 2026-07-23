package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type Parcel struct {
	ID          uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Name        string         `gorm:"type:text;not null" json:"name"`
	Description string         `gorm:"type:text" json:"description"`
	Geometry    string         `gorm:"type:geometry(Polygon,4326);not null" json:"geometry"`
	AreaHa      float64        `gorm:"->;type:double precision" json:"area_ha"` // Read-only, computed by PostGIS
	Metadata    datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'" json:"metadata"`
	CreatedAt   time.Time      `gorm:"type:timestamptz;not null;default:now()" json:"created_at"`
	UpdatedAt   time.Time      `gorm:"type:timestamptz;not null;default:now()" json:"updated_at"`
}
