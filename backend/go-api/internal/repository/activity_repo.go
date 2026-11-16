package repository

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Activity kinds, newest first in the feed.
const (
	ActivityParcelRegistered  = "parcel_registered"
	ActivitySnapshotQueued    = "snapshot_queued"
	ActivitySnapshotCompleted = "snapshot_completed"
	ActivitySnapshotFailed    = "snapshot_failed"
	ActivitySeriesStarted     = "series_started"
	ActivitySeriesSealed      = "series_sealed"
)

// ActivityEvent is one line of the recent-activity feed. Fields that don't apply to a kind are omitted.
type ActivityEvent struct {
	Kind            string      `json:"kind"`
	At              time.Time   `json:"at"`
	ParcelID        uuid.UUID   `json:"parcel_id"`
	ParcelShortID   string      `json:"parcel_short_id"`
	ParcelName      string      `json:"parcel_name"`
	SnapshotShortID *string     `json:"snapshot_short_id,omitempty"`
	SeriesShortID   *string     `json:"series_short_id,omitempty"`
	RecordHash      *string     `json:"record_hash,omitempty"`
	NdviMean        *float64    `json:"ndvi_mean,omitempty"`
	Windows         *int        `json:"windows,omitempty"`
	Flags           jsonStrings `json:"flags,omitempty"`
	Detail          *string     `json:"detail,omitempty"`
}

type jsonStrings []string

func (s *jsonStrings) Scan(src any) error {
	switch v := src.(type) {
	case nil:
		*s = nil
		return nil
	case []byte:
		return json.Unmarshal(v, s)
	case string:
		return json.Unmarshal([]byte(v), s)
	}
	return fmt.Errorf("jsonStrings: unsupported type %T", src)
}

func (s jsonStrings) Value() (driver.Value, error) {
	b, err := json.Marshal([]string(s))
	return string(b), err
}

type ActivityRepository interface {
	ListForOwner(ownerID uuid.UUID, limit int) ([]ActivityEvent, error)
}

type activityRepository struct {
	db *gorm.DB
}

func NewActivityRepository(db *gorm.DB) ActivityRepository {
	return &activityRepository{db: db}
}

// Series windows appear only through their series' started and sealed events, except
// failures, so a 24-window series doesn't fill the feed. Queued shows only while a snapshot
// is still running; after that its completed or failed event says more.
const activityQuery = `
WITH owned AS (
	SELECT id, short_id, name, created_at FROM parcels WHERE owner_id = @owner
)
SELECT * FROM (
	SELECT 'parcel_registered' AS kind, p.created_at AS at,
		p.id AS parcel_id, p.short_id AS parcel_short_id, p.name AS parcel_name,
		NULL::text AS snapshot_short_id, NULL::text AS series_short_id, NULL::text AS record_hash,
		NULL::float8 AS ndvi_mean, NULL::int AS windows, NULL::jsonb AS flags, NULL::text AS detail
	FROM owned p

	UNION ALL
	SELECT 'snapshot_queued', s.created_at, p.id, p.short_id, p.name,
		s.short_id, NULL, NULL, NULL, NULL, NULL, NULL
	FROM snapshots s
	JOIN owned p ON p.id = s.parcel_id
	WHERE s.series_id IS NULL AND s.status IN ('pending', 'processing')

	UNION ALL
	SELECT 'snapshot_completed', s.completed_at, p.id, p.short_id, p.name,
		s.short_id, NULL, s.record_hash,
		(SELECT m.value::float8
		 FROM measurements m
		 JOIN measurement_types mt ON mt.id = m.measurement_type_id
		 WHERE m.snapshot_id = s.id AND mt.slug = 'ndvi_mean'),
		NULL,
		(SELECT jsonb_agg(f.flag ORDER BY f.flag) FROM snapshot_flags f WHERE f.snapshot_id = s.id),
		NULL
	FROM snapshots s
	JOIN owned p ON p.id = s.parcel_id
	WHERE s.status = 'completed' AND s.series_id IS NULL

	UNION ALL
	SELECT 'snapshot_failed', COALESCE(j.updated_at, s.created_at), p.id, p.short_id, p.name,
		s.short_id, r.short_id, NULL, NULL, NULL, NULL, s.failure_message
	FROM snapshots s
	JOIN owned p ON p.id = s.parcel_id
	LEFT JOIN snapshot_jobs j ON j.snapshot_id = s.id
	LEFT JOIN snapshot_series r ON r.id = s.series_id
	WHERE s.status = 'failed'

	UNION ALL
	SELECT 'series_started', r.created_at, p.id, p.short_id, p.name,
		NULL, r.short_id, NULL, NULL,
		(SELECT count(*) FROM snapshots w WHERE w.series_id = r.id)::int, NULL, NULL
	FROM snapshot_series r
	JOIN owned p ON p.id = r.parcel_id

	UNION ALL
	SELECT 'series_sealed', r.sealed_at, p.id, p.short_id, p.name,
		NULL, r.short_id, r.record_hash, NULL,
		(SELECT count(*) FROM snapshots w WHERE w.series_id = r.id)::int, NULL, NULL
	FROM snapshot_series r
	JOIN owned p ON p.id = r.parcel_id
	WHERE r.sealed_at IS NOT NULL
) events
ORDER BY at DESC
LIMIT @limit`

func (r *activityRepository) ListForOwner(ownerID uuid.UUID, limit int) ([]ActivityEvent, error) {
	events := []ActivityEvent{}
	err := r.db.Raw(activityQuery, map[string]any{"owner": ownerID, "limit": limit}).Scan(&events).Error
	return events, err
}
