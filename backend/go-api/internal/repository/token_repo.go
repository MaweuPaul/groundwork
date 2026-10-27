package repository

import (
	"errors"
	"time"

	"groundwork/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RefreshTokenRepository interface {
	Create(token *models.RefreshToken) error
	GetByHash(hash string) (*models.RefreshToken, error)
	Revoke(id uuid.UUID) error
}

// the worker: holds the database connection
type refreshTokenRepository struct {
	db *gorm.DB
}

// hiring: build the worker, hand it back as "a RefreshTokenRepository"
func NewRefreshTokenRepository(db *gorm.DB) RefreshTokenRepository {
	return &refreshTokenRepository{db: db}
}

// save a new token row
func (r *refreshTokenRepository) Create(token *models.RefreshToken) error {
	return r.db.Create(token).Error
}

// find a token by its hash; nil, nil if it doesn't exist
func (r *refreshTokenRepository) GetByHash(hash string) (*models.RefreshToken, error) {
	var token models.RefreshToken
	err := r.db.Where("token_hash = ?", hash).First(&token).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &token, nil
}

// mark a token as used
func (r *refreshTokenRepository) Revoke(id uuid.UUID) error {
	return r.db.Model(&models.RefreshToken{}).
		Where("id = ?", id).
		Update("revoked_at", time.Now()).Error
}
