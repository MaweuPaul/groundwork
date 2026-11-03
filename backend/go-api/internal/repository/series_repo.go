package repository

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// NewSeries is a series request already split into windows by the service.
type NewSeries struct {
	Template       NewSnapshot // everything but the dates
	DateStart      time.Time
	DateEnd        time.Time
	WindowMonths   int
	IntervalMonths int
	Windows        [][2]time.Time
}

type SeriesRef struct {
	ID      uuid.UUID `json:"id"`
	ShortID string    `json:"short_id"`
	Windows int       `json:"windows"`
}

// SeriesSummary is one series with its progress and, once every window finished, its seal.
type SeriesSummary struct {
	ID                  uuid.UUID  `json:"id"`
	ShortID             string     `json:"short_id"`
	DateStart           string     `json:"date_start"`
	DateEnd             string     `json:"date_end"`
	WindowMonths        int        `json:"window_months"`
	IntervalMonths      int        `json:"interval_months"`
	MaxSceneCloudPct    float64    `json:"max_scene_cloud_pct"`
	CloudScoreThreshold float64    `json:"cloud_score_threshold"`
	Windows             int        `json:"windows"`
	Completed           int        `json:"completed"`
	Failed              int        `json:"failed"`
	RecordHash          *string    `json:"record_hash,omitempty"`
	SealedAt            *time.Time `json:"sealed_at,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
}

// CreateSeriesWithJobs inserts the series, its window snapshots and their jobs in one
// transaction, so a series never exists half-queued.
func (r *snapshotRepository) CreateSeriesWithJobs(in NewSeries) (SeriesRef, error) {
	var ref SeriesRef
	t := in.Template
	err := r.db.Transaction(func(tx *gorm.DB) error {
		res := tx.Raw(`
			INSERT INTO snapshot_series (parcel_id, methodology_id, methodology_version, date_start, date_end,
			                             window_months, interval_months, max_scene_cloud_pct, cloud_score_threshold)
			SELECT p.id, ?, ?, ?::date, ?::date, ?, ?, ?, ?
			FROM parcels p
			WHERE p.id = ? AND p.owner_id = ?
			RETURNING id, short_id`,
			t.MethodologyID, t.MethodologyVersion,
			in.DateStart.Format(time.DateOnly), in.DateEnd.Format(time.DateOnly),
			in.WindowMonths, in.IntervalMonths, t.MaxSceneCloudPct, t.CloudScoreThreshold,
			t.ParcelID, t.OwnerID,
		).Scan(&ref)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrParcelNotFound
		}

		ref.Windows = len(in.Windows)
		for _, w := range in.Windows {
			snap := t
			snap.DateStart, snap.DateEnd = w[0], w[1]
			if _, err := insertSnapshotWithJob(tx, snap, &ref.ID); err != nil {
				return err
			}
		}
		return nil
	})
	return ref, err
}

// ListSeriesByParcelForOwner returns ErrParcelNotFound for someone else's parcel.
func (r *snapshotRepository) ListSeriesByParcelForOwner(parcelID, ownerID uuid.UUID) ([]SeriesSummary, error) {
	var owned bool
	if err := r.db.Raw(`SELECT EXISTS (SELECT 1 FROM parcels WHERE id = ? AND owner_id = ?)`, parcelID, ownerID).
		Scan(&owned).Error; err != nil {
		return nil, err
	}
	if !owned {
		return nil, ErrParcelNotFound
	}

	series := []SeriesSummary{}
	err := r.db.Raw(`
		SELECT r.id, r.short_id,
		       to_char(r.date_start, 'YYYY-MM-DD') AS date_start,
		       to_char(r.date_end, 'YYYY-MM-DD') AS date_end,
		       r.window_months, r.interval_months, r.max_scene_cloud_pct, r.cloud_score_threshold,
		       count(s.id) AS windows,
		       count(s.id) FILTER (WHERE s.status = 'completed') AS completed,
		       count(s.id) FILTER (WHERE s.status = 'failed') AS failed,
		       r.record_hash, r.sealed_at, r.created_at
		FROM snapshot_series r
		LEFT JOIN snapshots s ON s.series_id = r.id
		WHERE r.parcel_id = ?
		GROUP BY r.id
		ORDER BY r.created_at DESC`,
		parcelID,
	).Scan(&series).Error
	return series, err
}
