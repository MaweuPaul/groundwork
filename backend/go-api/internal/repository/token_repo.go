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
	Revoke(id uuid.UUID) (bool, error)
	RevokeForLogout(id uuid.UUID) error
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

// mark a token as used; false if it was already revoked. The check and the
// update are one statement, so two concurrent refreshes can't both revoke it.
func (r *refreshTokenRepository) Revoke(id uuid.UUID) (bool, error) {
	res := r.db.Model(&models.RefreshToken{}).
		Where("id = ? AND revoked_at IS NULL", id).
		Updates(map[string]any{"revoked_at": time.Now(), "revoke_reason": "rotated"})
	return res.RowsAffected == 1, res.Error
}

// end a session; also ends the reuse window of a token that was just rotated
func (r *refreshTokenRepository) RevokeForLogout(id uuid.UUID) error {
	return r.db.Model(&models.RefreshToken{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"revoked_at":    gorm.Expr("COALESCE(revoked_at, NOW())"),
			"revoke_reason": "logout",
		}).Error
}
