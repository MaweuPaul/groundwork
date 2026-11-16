package service

import (
	"github.com/google/uuid"

	"groundwork/internal/repository"
)

const (
	DefaultActivityLimit = 10
	MaxActivityLimit     = 50
)

type ActivityService interface {
	Recent(ownerID uuid.UUID, limit int) ([]repository.ActivityEvent, error)
}

type activityService struct {
	repo repository.ActivityRepository
}

func NewActivityService(repo repository.ActivityRepository) ActivityService {
	return &activityService{repo: repo}
}

// Recent returns the owner's latest events, newest first. limit is clamped to 1..MaxActivityLimit.
func (s *activityService) Recent(ownerID uuid.UUID, limit int) ([]repository.ActivityEvent, error) {
	if limit <= 0 {
		limit = DefaultActivityLimit
	}
	return s.repo.ListForOwner(ownerID, min(limit, MaxActivityLimit))
}
