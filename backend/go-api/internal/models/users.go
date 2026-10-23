package models

import (
	"github.com/google/uuid"
)

type User struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Email     string    `gorm:"type:text;not null;unique" json:"email"`
	Password  string    `gorm:"type:text;not null" json:"-"`
	firstName string    `gorm:"type:text;not null" json:"first_name"`
	lastName  string    `gorm:"type:text;not null" json:"last_name"`
	username  string    `gorm:"type:text;not null;unique" json:"username"`
}
