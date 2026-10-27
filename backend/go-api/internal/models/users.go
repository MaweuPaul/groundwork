package models

import (
	"time"

	"github.com/google/uuid"
)

type UserRole string

const (
	RoleAuditor     UserRole = "auditor"
	RoleAssetHolder UserRole = "asset_holder"
)

func (r UserRole) Valid() bool {
	return r == RoleAuditor || r == RoleAssetHolder
}

type User struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Email     string    `gorm:"type:text;not null;unique" json:"email"`
	Password  string    `gorm:"type:text;not null" json:"-"`
	FirstName string    `gorm:"type:text;not null" json:"first_name"`
	LastName  string    `gorm:"type:text;not null" json:"last_name"`
	Username  string    `gorm:"type:text;not null;unique" json:"username"`
	Role      UserRole  `gorm:"type:user_role;not null" json:"role"`
	CreatedAt time.Time `gorm:"type:timestamptz;not null;default:now()" json:"created_at"`
	UpdatedAt time.Time `gorm:"type:timestamptz;not null;default:now()" json:"updated_at"`
}
