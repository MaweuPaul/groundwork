package service

import (
	"errors"
	"fmt"
	"sort"
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

type CreateSnapshotInput struct {
	ParcelID           uuid.UUID
	OwnerID            uuid.UUID
	Methodology        string // slug, e.g. "ndvi-s2-median"
	MethodologyVersion string // required; never defaults to the latest
	DateStart          string // YYYY-MM-DD
	DateEnd            string // YYYY-MM-DD
	// Optional; any the methodology takes but the request leaves out get their defaults.
	Parameters map[string]float64
}

type SnapshotService interface {
	CreateSnapshot(in CreateSnapshotInput) (uuid.UUID, error)
	GetSnapshot(id, ownerID uuid.UUID) (*repository.SnapshotDetail, error)
	GetNdviImage(id, ownerID uuid.UUID) ([]byte, error)
	ListParcelSnapshots(parcelID, ownerID uuid.UUID) ([]repository.SnapshotSummary, error)
	ListMethodologies() ([]models.Methodology, error)
	CreateSeries(in CreateSeriesInput) (repository.SeriesRef, error)
	ListParcelSeries(parcelID, ownerID uuid.UUID) ([]repository.SeriesSummary, error)
}

type snapshotService struct {
	snapshots     repository.SnapshotRepository
	methodologies repository.MethodologyRepository
	limits        repository.Limits
	now           func() time.Time
}

func NewSnapshotService(snapshots repository.SnapshotRepository, methodologies repository.MethodologyRepository, limits repository.Limits) SnapshotService {
	return &snapshotService{snapshots: snapshots, methodologies: methodologies, limits: limits, now: time.Now}
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

	start, end, err := s.parseDates(in.DateStart, in.DateEnd, true)
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

	params, err := s.resolveParameters(m.ID, in.Parameters)
	if err != nil {
		return uuid.Nil, err
	}

	return s.snapshots.CreateWithJob(repository.NewSnapshot{
		ParcelID:           in.ParcelID,
		OwnerID:            in.OwnerID,
		MethodologyID:      m.ID,
		MethodologyVersion: m.Version,
		DateStart:          start,
		DateEnd:            end,
		Parameters:         params,
		Limits:             s.limits,
	})
}

// resolveParameters checks the request's parameters against the methodology's and fills in
// defaults. The database checks the same rules on insert; this gives the client a clear message.
func (s *snapshotService) resolveParameters(methodologyID uuid.UUID, given map[string]float64) (repository.Parameters, error) {
	specs, err := s.methodologies.Parameters(methodologyID)
	if err != nil {
		return nil, err
	}

	known := make(map[string]bool, len(specs))
	for _, spec := range specs {
		known[spec.Name] = true
	}
	var unknown []string
	for name := range given {
		if !known[name] {
			unknown = append(unknown, name)
		}
	}
	if len(unknown) > 0 {
		sort.Strings(unknown)
		return nil, &ValidationError{fmt.Sprintf("unknown parameter %q for this methodology", unknown[0])}
	}

	resolved := make(repository.Parameters, len(specs))
	for _, spec := range specs {
		value, ok := given[spec.Name]
		if !ok {
			value = spec.Default
		}
		if value < spec.Min || value > spec.Max {
			return nil, &ValidationError{fmt.Sprintf("%s must be between %g and %g", spec.Name, spec.Min, spec.Max)}
		}
		resolved[spec.Name] = value
	}
	return resolved, nil
}

// limitRange caps the range at a year; a series range can be longer because each window is short.
func (s *snapshotService) parseDates(rawStart, rawEnd string, limitRange bool) (time.Time, time.Time, error) {
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
	case limitRange && end.Sub(start) > maxSnapshotRange:
		return time.Time{}, time.Time{}, &ValidationError{"the date range cannot be longer than one year"}
	}

	return start, end, nil
}
