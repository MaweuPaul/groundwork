package service

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"groundwork/internal/repository"

	"github.com/google/uuid"
)

// ErrUnknownMethodology means no methodology has that slug and version.
var ErrUnknownMethodology = errors.New("unknown methodology or version")

// ValidationError is a problem with the request the client can fix; Message is safe to show.
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string { return e.Message }

// sentinel2Start is the first day of Sentinel-2 imagery.
var sentinel2Start = time.Date(2015, 6, 23, 0, 0, 0, 0, time.UTC)

// maxSnapshotRange keeps one Earth Engine job to at most a year of imagery.
const maxSnapshotRange = 366 * 24 * time.Hour

type CreateSnapshotInput struct {
	ParcelID           uuid.UUID
	OwnerID            uuid.UUID
	Methodology        string // slug, e.g. "ndvi-s2-median"
	MethodologyVersion string // required; never defaults to the latest
	DateStart          string // YYYY-MM-DD
	DateEnd            string // YYYY-MM-DD
}

type SnapshotService interface {
	CreateSnapshot(in CreateSnapshotInput) (uuid.UUID, error)
	GetSnapshot(id, ownerID uuid.UUID) (*repository.SnapshotDetail, error)
}

type snapshotService struct {
	snapshots     repository.SnapshotRepository
	methodologies repository.MethodologyRepository
	now           func() time.Time
}

func NewSnapshotService(snapshots repository.SnapshotRepository, methodologies repository.MethodologyRepository) SnapshotService {
	return &snapshotService{snapshots: snapshots, methodologies: methodologies, now: time.Now}
}

// GetSnapshot returns nil, nil when the snapshot doesn't exist or isn't the owner's.
func (s *snapshotService) GetSnapshot(id, ownerID uuid.UUID) (*repository.SnapshotDetail, error) {
	return s.snapshots.GetByIDForOwner(id, ownerID)
}

// CreateSnapshot validates the request and queues the snapshot. The job is picked
// up by the Python worker; the caller gets the snapshot id straight away.
func (s *snapshotService) CreateSnapshot(in CreateSnapshotInput) (uuid.UUID, error) {
	slug := strings.TrimSpace(in.Methodology)
	version := strings.TrimSpace(in.MethodologyVersion)
	if slug == "" || version == "" {
		return uuid.Nil, &ValidationError{"methodology and methodology_version are required"}
	}

	start, end, err := s.parseDates(in.DateStart, in.DateEnd)
	if err != nil {
		return uuid.Nil, err
	}

	m, err := s.methodologies.GetBySlugVersion(slug, version)
	if err != nil {
		return uuid.Nil, err
	}
	if m == nil {
		return uuid.Nil, ErrUnknownMethodology
	}

	return s.snapshots.CreateWithJob(repository.NewSnapshot{
		ParcelID:           in.ParcelID,
		OwnerID:            in.OwnerID,
		MethodologyID:      m.ID,
		MethodologyVersion: m.Version,
		DateStart:          start,
		DateEnd:            end,
	})
}

func (s *snapshotService) parseDates(rawStart, rawEnd string) (time.Time, time.Time, error) {
	start, err := time.Parse(time.DateOnly, strings.TrimSpace(rawStart))
	if err != nil {
		return time.Time{}, time.Time{}, &ValidationError{"date_start must be a date in YYYY-MM-DD format"}
	}
	end, err := time.Parse(time.DateOnly, strings.TrimSpace(rawEnd))
	if err != nil {
		return time.Time{}, time.Time{}, &ValidationError{"date_end must be a date in YYYY-MM-DD format"}
	}

	now := s.now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	switch {
	case end.Before(start):
		return time.Time{}, time.Time{}, &ValidationError{"date_end must be on or after date_start"}
	case end.After(today):
		return time.Time{}, time.Time{}, &ValidationError{"date_end cannot be in the future"}
	case start.Before(sentinel2Start):
		return time.Time{}, time.Time{}, &ValidationError{
			fmt.Sprintf("date_start cannot be before %s, when Sentinel-2 imagery begins", sentinel2Start.Format(time.DateOnly)),
		}
	case end.Sub(start) > maxSnapshotRange:
		return time.Time{}, time.Time{}, &ValidationError{"the date range cannot be longer than one year"}
	}

	return start, end, nil
}
