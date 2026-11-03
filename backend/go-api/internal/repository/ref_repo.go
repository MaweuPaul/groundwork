package repository

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// RefRepository looks rows up by their short public ID (see migration 008).
type RefRepository interface {
	ParcelIDByShortID(shortID string) (uuid.UUID, bool, error)
	SnapshotIDByShortID(shortID string) (uuid.UUID, bool, error)
}

type refRepository struct {
	db *gorm.DB
}

func NewRefRepository(db *gorm.DB) RefRepository {
	return &refRepository{db: db}
}

func (r *refRepository) ParcelIDByShortID(shortID string) (uuid.UUID, bool, error) {
	return r.idByShortID(`SELECT id FROM parcels WHERE short_id = ?`, shortID)
}

func (r *refRepository) SnapshotIDByShortID(shortID string) (uuid.UUID, bool, error) {
	return r.idByShortID(`SELECT id FROM snapshots WHERE short_id = ?`, shortID)
}

func (r *refRepository) idByShortID(query, shortID string) (uuid.UUID, bool, error) {
	var ids []uuid.UUID
	if err := r.db.Raw(query, shortID).Scan(&ids).Error; err != nil {
		return uuid.Nil, false, err
	}
	if len(ids) == 0 {
		return uuid.Nil, false, nil
	}
	return ids[0], true, nil
}
