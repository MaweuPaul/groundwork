package repository

import (
	"errors"

	"groundwork/internal/models"

	"gorm.io/gorm"
)

type UserRepository interface {
	Create(user *models.User) error
	GetByEmail(email string) (*models.User, error)
	GetByUsername(username string) (*models.User, error)
}

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(user *models.User) error {
	return r.db.Create(user).Error
}

// GetByEmail returns nil, nil when no user has that email.
func (r *userRepository) GetByEmail(email string) (*models.User, error) {
	return r.getBy("email", email)
}

// GetByUsername returns nil, nil when no user has that username.
func (r *userRepository) GetByUsername(username string) (*models.User, error) {
	return r.getBy("username", username)
}

// column is always a constant from this file, never user input.
func (r *userRepository) getBy(column, value string) (*models.User, error) {
	var user models.User
	err := r.db.Where(column+" = ?", value).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}
