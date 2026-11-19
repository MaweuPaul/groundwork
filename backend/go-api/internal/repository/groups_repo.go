package repository

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// NewGroup is several methodologies run on one parcel for the same window. Each member
// carries its own methodology and parameters; parcel, owner and dates come from the group.
type NewGroup struct {
	ParcelID  uuid.UUID
	OwnerID   uuid.UUID
	DateStart time.Time
	DateEnd   time.Time
	Members   []NewSnapshot
	Limits    Limits
}

type GroupRef struct {
	ID      uuid.UUID `json:"id"`
	ShortID string    `json:"short_id"`
	Members int       `json:"members"`
}

// GroupSummary is one group with its members' progress and results and, once every member
// finished, its seal.
type GroupSummary struct {
	ID         uuid.UUID     `json:"id"`
	ShortID    string        `json:"short_id"`
	DateStart  string        `json:"date_start"`
	DateEnd    string        `json:"date_end"`
	RecordHash *string       `json:"record_hash,omitempty"`
	SealedAt   *time.Time    `json:"sealed_at,omitempty"`
	CreatedAt  time.Time     `json:"created_at"`
	Members    []GroupMember `json:"members" gorm:"-"`
}

type GroupMember struct {
	SnapshotID         uuid.UUID          `json:"snapshot_id"`
	ShortID            string             `json:"short_id"`
	Methodology        string             `json:"methodology"` // slug
	MethodologyVersion string             `json:"methodology_version"`
	Status             string             `json:"status"`
	RecordHash         *string            `json:"record_hash,omitempty"`
	Measurements       map[string]float64 `json:"measurements" gorm:"-"`
}

// CreateGroupWithJobs inserts the group, one snapshot per member and their jobs in one
// transaction, so a group never exists half-queued.
func (r *snapshotRepository) CreateGroupWithJobs(in NewGroup) (GroupRef, error) {
	var ref GroupRef
	err := r.db.Transaction(func(tx *gorm.DB) error {
		if err := lockOwner(tx, in.OwnerID); err != nil {
			return err
		}
		if err := checkParcel(tx, in.ParcelID, in.OwnerID, in.Limits); err != nil {
			return err
		}
		if err := checkQuota(tx, in.OwnerID, in.Limits, len(in.Members)); err != nil {
			return err
		}

		res := tx.Raw(`
			INSERT INTO snapshot_groups (parcel_id, date_start, date_end)
			SELECT p.id, ?::date, ?::date
			FROM parcels p
			WHERE p.id = ? AND p.owner_id = ?
			RETURNING id, short_id`,
			in.DateStart.Format(time.DateOnly), in.DateEnd.Format(time.DateOnly),
			in.ParcelID, in.OwnerID,
		).Scan(&ref)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrParcelNotFound
		}

		ref.Members = len(in.Members)
		for _, m := range in.Members {
			m.ParcelID, m.OwnerID, m.DateStart, m.DateEnd = in.ParcelID, in.OwnerID, in.DateStart, in.DateEnd
			if _, err := insertSnapshotWithJob(tx, m, membership{GroupID: &ref.ID}); err != nil {
				return err
			}
		}
		return nil
	})
	return ref, err
}

// ListGroupsByParcelForOwner returns ErrParcelNotFound for someone else's parcel.
func (r *snapshotRepository) ListGroupsByParcelForOwner(parcelID, ownerID uuid.UUID) ([]GroupSummary, error) {
	if err := requireOwnedParcel(r.db, parcelID, ownerID); err != nil {
		return nil, err
	}

	groups := []GroupSummary{}
	err := r.db.Raw(`
		SELECT id, short_id,
		       to_char(date_start, 'YYYY-MM-DD') AS date_start,
		       to_char(date_end, 'YYYY-MM-DD') AS date_end,
		       record_hash, sealed_at, created_at
		FROM snapshot_groups
		WHERE parcel_id = ?
		ORDER BY created_at DESC`,
		parcelID,
	).Scan(&groups).Error
	if err != nil || len(groups) == 0 {
		return groups, err
	}

	ids := make([]uuid.UUID, len(groups))
	index := make(map[uuid.UUID]int, len(groups))
	for i, g := range groups {
		ids[i], index[g.ID] = g.ID, i
	}

	var members []struct {
		GroupID uuid.UUID
		GroupMember
	}
	err = r.db.Raw(`
		SELECT s.group_id, s.id AS snapshot_id, s.short_id, m.slug AS methodology,
		       s.methodology_version, s.status, s.record_hash
		FROM snapshots s
		JOIN methodologies m ON m.id = s.methodology_id
		WHERE s.group_id IN ?
		ORDER BY m.slug, s.methodology_version`,
		ids,
	).Scan(&members).Error
	if err != nil {
		return nil, err
	}

	snapshotIDs := make([]uuid.UUID, len(members))
	for i, m := range members {
		snapshotIDs[i] = m.SnapshotID
	}
	measurements, err := measurementsBySnapshot(r.db, snapshotIDs)
	if err != nil {
		return nil, err
	}

	for _, m := range members {
		member := m.GroupMember
		member.Measurements = measurements[member.SnapshotID]
		if member.Measurements == nil {
			member.Measurements = map[string]float64{}
		}
		g := &groups[index[m.GroupID]]
		g.Members = append(g.Members, member)
	}
	return groups, nil
}

// requireOwnedParcel returns ErrParcelNotFound unless the parcel exists and is the owner's.
func requireOwnedParcel(db *gorm.DB, parcelID, ownerID uuid.UUID) error {
	var owned bool
	if err := db.Raw(`SELECT EXISTS (SELECT 1 FROM parcels WHERE id = ? AND owner_id = ?)`, parcelID, ownerID).
		Scan(&owned).Error; err != nil {
		return err
	}
	if !owned {
		return ErrParcelNotFound
	}
	return nil
}
