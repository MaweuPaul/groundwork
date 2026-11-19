package repository

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ChainCheck is one parcel's sealed records recomputed from what is stored now.
type ChainCheck struct {
	Intact    bool           `json:"intact"`
	Records   int            `json:"records"`
	Groups    int            `json:"groups"`
	HeadSeq   *int64         `json:"head_seq"`
	HeadHash  *string        `json:"head_hash"`
	Problems  []ChainProblem `json:"problems"`
	CheckedAt time.Time      `json:"checked_at"`
}

// ChainProblem names the record or group that failed; a broken chain head has neither.
type ChainProblem struct {
	Seq          *int64     `json:"seq,omitempty"`
	SnapshotID   *uuid.UUID `json:"snapshot_id,omitempty"`
	ShortID      *string    `json:"short_id,omitempty"`
	GroupShortID *string    `json:"group_short_id,omitempty"`
	Problem      string     `json:"problem"`
}

// VerifyParcelForOwner rechecks every completed record in the parcel's chain and every sealed
// group. It reads one consistent snapshot of the database, so a record sealed mid-check can't
// show up as a problem. Returns ErrParcelNotFound for someone else's parcel.
func (r *snapshotRepository) VerifyParcelForOwner(parcelID, ownerID uuid.UUID) (*ChainCheck, error) {
	check := &ChainCheck{Problems: []ChainProblem{}}
	err := r.db.Transaction(func(tx *gorm.DB) error {
		if err := requireOwnedParcel(tx, parcelID, ownerID); err != nil {
			return err
		}

		if err := tx.Raw(`
			SELECT v.bad_seq AS seq, v.bad_snapshot_id AS snapshot_id, s.short_id,
			       NULL::text AS group_short_id, v.problem
			FROM verify_parcel_chain(?) v
			LEFT JOIN snapshots s ON s.id = v.bad_snapshot_id
			UNION ALL
			SELECT NULL::bigint, NULL::uuid, NULL::text, g.short_id, v.problem
			FROM snapshot_groups g
			CROSS JOIN LATERAL verify_snapshot_group(g.id) v
			WHERE g.parcel_id = ?
			ORDER BY seq NULLS LAST`,
			parcelID, parcelID,
		).Scan(&check.Problems).Error; err != nil {
			return err
		}

		var counts struct {
			Records   int
			Groups    int
			HeadSeq   *int64
			HeadHash  *string
			CheckedAt time.Time
		}
		if err := tx.Raw(`
			SELECT (SELECT count(*) FROM snapshots WHERE parcel_id = ? AND status = 'completed') AS records,
			       (SELECT count(*) FROM snapshot_groups WHERE parcel_id = ? AND record_hash IS NOT NULL) AS groups,
			       h.seq AS head_seq, h.hash AS head_hash, now() AS checked_at
			FROM (SELECT 1) one
			LEFT JOIN parcel_chain_heads h ON h.parcel_id = ?`,
			parcelID, parcelID, parcelID,
		).Scan(&counts).Error; err != nil {
			return err
		}
		check.Records, check.Groups = counts.Records, counts.Groups
		check.HeadSeq, check.HeadHash, check.CheckedAt = counts.HeadSeq, counts.HeadHash, counts.CheckedAt
		return nil
	}, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return nil, err
	}

	check.Intact = len(check.Problems) == 0
	return check, nil
}
