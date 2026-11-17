package repository

import (
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Limits cap the Earth Engine work one owner can queue, since every snapshot spends the
// project's compute quota. Zero means no limit.
type Limits struct {
	// Snapshots, each series window counted, created in any 24 hours.
	PerDay int
	// The largest parcel a snapshot may cover; Earth Engine's cost grows with the area.
	MaxAreaHa float64
}

// LimitError is a limit the request ran into; Message is safe to show. Quota is true when
// it frees up with time (the daily cap), false when the request itself is too big.
type LimitError struct {
	Message string
	Quota   bool
}

func (e *LimitError) Error() string { return e.Message }

// lockOwner serialises snapshot requests from one owner until the transaction ends, so two
// sent at once (a double click) see each other in the duplicate and quota checks.
func lockOwner(tx *gorm.DB, ownerID uuid.UUID) error {
	return tx.Exec(`SELECT pg_advisory_xact_lock(hashtextextended(?, 0))`, ownerID.String()).Error
}

// checkParcel confirms the parcel is the owner's and within the area limit.
func checkParcel(tx *gorm.DB, parcelID, ownerID uuid.UUID, l Limits) error {
	var rows []struct{ AreaHa *float64 }
	if err := tx.Raw(`SELECT area_ha FROM parcels WHERE id = ? AND owner_id = ?`, parcelID, ownerID).
		Scan(&rows).Error; err != nil {
		return err
	}
	if len(rows) == 0 {
		return ErrParcelNotFound
	}
	if area := rows[0].AreaHa; l.MaxAreaHa > 0 && area != nil && *area > l.MaxAreaHa {
		return &LimitError{Message: fmt.Sprintf(
			"This parcel is %s ha. Snapshots are limited to parcels of %s ha or less, to keep within the Earth Engine quota.",
			hectares(*area), hectares(l.MaxAreaHa),
		)}
	}
	return nil
}

// checkQuota refuses a request that would take the owner past the daily cap.
func checkQuota(tx *gorm.DB, ownerID uuid.UUID, l Limits, needed int) error {
	if l.PerDay <= 0 {
		return nil
	}
	var used struct {
		N        int
		FreesInS *float64
	}
	if err := tx.Raw(`
		SELECT count(*) AS n,
		       extract(epoch FROM min(s.created_at) + interval '24 hours' - now()) AS frees_in_s
		FROM snapshots s
		JOIN parcels p ON p.id = s.parcel_id
		WHERE p.owner_id = ? AND s.created_at > now() - interval '24 hours'`,
		ownerID,
	).Scan(&used).Error; err != nil {
		return err
	}
	if used.N+needed <= l.PerDay {
		return nil
	}

	left := max(l.PerDay-used.N, 0)
	if needed > 1 && left > 0 {
		return &LimitError{Quota: true, Message: fmt.Sprintf(
			"This series needs %d snapshots, but only %d of your %d for the last 24 hours are left.",
			needed, left, l.PerDay,
		)}
	}
	wait := "later"
	if used.FreesInS != nil {
		wait = "in " + duration(time.Duration(*used.FreesInS)*time.Second)
	}
	return &LimitError{Quota: true, Message: fmt.Sprintf(
		"You've used all %d snapshots for the last 24 hours. The next one frees up %s.", l.PerDay, wait,
	)}
}

func hectares(v float64) string {
	s := strconv.FormatFloat(math.Round(v), 'f', 0, 64)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}

func duration(d time.Duration) string {
	if d < time.Hour {
		m := max(int(math.Ceil(d.Minutes())), 1)
		if m == 1 {
			return "1 minute"
		}
		return fmt.Sprintf("%d minutes", m)
	}
	h := int(math.Ceil(d.Hours()))
	if h == 1 {
		return "1 hour"
	}
	return fmt.Sprintf("%d hours", h)
}
