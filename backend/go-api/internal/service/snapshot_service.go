package service

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"groundwork/internal/models"
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

// Cloud Score+ cs_cdf a pixel needs to count as clear, unless the request sets one.
// Google recommends 0.50 to 0.65; outside 0.40 to 0.80 results stop being useful.
const (
	defaultCloudScoreThreshold = 0.60
	minCloudScoreThreshold     = 0.40
	maxCloudScoreThreshold     = 0.80
)

type CreateSnapshotInput struct {
	ParcelID           uuid.UUID
	OwnerID            uuid.UUID
	Methodology        string // slug, e.g. "ndvi-s2-median"
	MethodologyVersion string // required; never defaults to the latest
	DateStart          string // YYYY-MM-DD
	DateEnd            string // YYYY-MM-DD
	// Optional; nil uses the methodology's max_cloud_pct and the default threshold.
	MaxSceneCloudPct    *float64
	CloudScoreThreshold *float64
}

type SnapshotService interface {
	CreateSnapshot(in CreateSnapshotInput) (uuid.UUID, error)
	GetSnapshot(id, ownerID uuid.UUID) (*repository.SnapshotDetail, error)
	GetNdviImage(id, ownerID uuid.UUID) ([]byte, error)
	ListParcelSnapshots(parcelID, ownerID uuid.UUID) ([]repository.SnapshotSummary, error)
	ListMethodologies() ([]models.Methodology, error)
}

type snapshotService struct {
	snapshots     repository.SnapshotRepository
	methodologies repository.MethodologyRepository
	now           func() time.Time
}

func NewSnapshotService(snapshots repository.SnapshotRepository, methodologies repository.MethodologyRepository) SnapshotService {
	return &snapshotService{snapshots: snapshots, methodologies: methodologies, now: time.Now}
}

// GetNdviImage returns the snapshot's NDVI map PNG, or nil when there is none or it isn't the owner's.
func (s *snapshotService) GetNdviImage(id, ownerID uuid.UUID) ([]byte, error) {
	return s.snapshots.NdviPNGForOwner(id, ownerID)
}

// GetSnapshot returns nil, nil when the snapshot doesn't exist or isn't the owner's.
func (s *snapshotService) GetSnapshot(id, ownerID uuid.UUID) (*repository.SnapshotDetail, error) {
	return s.snapshots.GetByIDForOwner(id, ownerID)
}

// ListParcelSnapshots returns repository.ErrParcelNotFound for someone else's parcel.
func (s *snapshotService) ListParcelSnapshots(parcelID, ownerID uuid.UUID) ([]repository.SnapshotSummary, error) {
	return s.snapshots.ListByParcelForOwner(parcelID, ownerID)
}

func (s *snapshotService) ListMethodologies() ([]models.Methodology, error) {
	return s.methodologies.List()
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

	cloudPct, threshold, err := cloudOptions(in, m.MaxCloudPct)
	if err != nil {
		return uuid.Nil, err
	}

	return s.snapshots.CreateWithJob(repository.NewSnapshot{
		ParcelID:            in.ParcelID,
		OwnerID:             in.OwnerID,
		MethodologyID:       m.ID,
		MethodologyVersion:  m.Version,
		DateStart:           start,
		DateEnd:             end,
		MaxSceneCloudPct:    cloudPct,
		CloudScoreThreshold: threshold,
	})
}

// cloudOptions applies the defaults and checks the ranges. Values are rounded the way
// the database stores them (2 and 3 decimals), so the sealed record matches the request.
func cloudOptions(in CreateSnapshotInput, methodologyDefault float64) (float64, float64, error) {
	cloudPct := methodologyDefault
	if in.MaxSceneCloudPct != nil {
		cloudPct = math.Round(*in.MaxSceneCloudPct*100) / 100
	}
	if cloudPct <= 0 || cloudPct > 100 {
		return 0, 0, &ValidationError{"max_scene_cloud_pct must be greater than 0 and at most 100"}
	}

	threshold := defaultCloudScoreThreshold
	if in.CloudScoreThreshold != nil {
		threshold = math.Round(*in.CloudScoreThreshold*1000) / 1000
	}
	if threshold < minCloudScoreThreshold || threshold > maxCloudScoreThreshold {
		return 0, 0, &ValidationError{
			fmt.Sprintf("cloud_score_threshold must be between %.2f and %.2f", minCloudScoreThreshold, maxCloudScoreThreshold),
		}
	}

	return cloudPct, threshold, nil
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
