package service

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"groundwork/internal/repository"
)

// maxSeriesWindows keeps one request to a reasonable number of Earth Engine jobs.
const maxSeriesWindows = 24

type CreateSeriesInput struct {
	CreateSnapshotInput // DateStart and DateEnd are the whole range
	WindowMonths        int
	IntervalMonths      int
}

// SeriesWindows splits [start, end] into windows of windowMonths, one starting every
// intervalMonths. A last window that would run past end is cut short to end on it.
func SeriesWindows(start, end time.Time, windowMonths, intervalMonths int) [][2]time.Time {
	var windows [][2]time.Time
	for i := 0; ; i++ {
		ws := start.AddDate(0, i*intervalMonths, 0)
		if ws.After(end) {
			return windows
		}
		we := ws.AddDate(0, windowMonths, -1)
		if we.After(end) {
			we = end
		}
		windows = append(windows, [2]time.Time{ws, we})
	}
}

func (s *snapshotService) CreateSeries(in CreateSeriesInput) (repository.SeriesRef, error) {
	if in.WindowMonths < 1 || in.WindowMonths > 12 || in.IntervalMonths < 1 || in.IntervalMonths > 12 {
		return repository.SeriesRef{}, &ValidationError{"window_months and interval_months must be between 1 and 12"}
	}
	slug := strings.TrimSpace(in.Methodology)
	version := strings.TrimSpace(in.MethodologyVersion)
	if slug == "" || version == "" {
		return repository.SeriesRef{}, &ValidationError{"methodology and methodology_version are required"}
	}

	start, end, err := s.parseDates(in.DateStart, in.DateEnd, false)
	if err != nil {
		return repository.SeriesRef{}, err
	}
	windows := SeriesWindows(start, end, in.WindowMonths, in.IntervalMonths)
	if len(windows) > maxSeriesWindows {
		return repository.SeriesRef{}, &ValidationError{
			fmt.Sprintf("this would create %d windows; the most is %d", len(windows), maxSeriesWindows),
		}
	}

	m, err := s.methodologies.GetBySlugVersion(slug, version)
	if err != nil {
		return repository.SeriesRef{}, err
	}
	if m == nil {
		return repository.SeriesRef{}, ErrUnknownMethodology
	}
	cloudPct, threshold, err := cloudOptions(in.CreateSnapshotInput, m.MaxCloudPct)
	if err != nil {
		return repository.SeriesRef{}, err
	}

	return s.snapshots.CreateSeriesWithJobs(repository.NewSeries{
		Template: repository.NewSnapshot{
			ParcelID:            in.ParcelID,
			OwnerID:             in.OwnerID,
			MethodologyID:       m.ID,
			MethodologyVersion:  m.Version,
			MaxSceneCloudPct:    cloudPct,
			CloudScoreThreshold: threshold,
		},
		DateStart:      start,
		DateEnd:        end,
		WindowMonths:   in.WindowMonths,
		IntervalMonths: in.IntervalMonths,
		Windows:        windows,
	})
}

// ListParcelSeries returns repository.ErrParcelNotFound for someone else's parcel.
func (s *snapshotService) ListParcelSeries(parcelID, ownerID uuid.UUID) ([]repository.SeriesSummary, error) {
	return s.snapshots.ListSeriesByParcelForOwner(parcelID, ownerID)
}
