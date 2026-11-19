package integration

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"groundwork/internal/repository"
)

func newGroup(t *testing.T, tx *gorm.DB, owner, parcelID uuid.UUID, slugs ...string) repository.GroupRef {
	t.Helper()
	members := make([]repository.NewSnapshot, len(slugs))
	for i, slug := range slugs {
		m := methodology(t, tx, slug)
		members[i] = repository.NewSnapshot{MethodologyID: m.ID, MethodologyVersion: m.Version, Parameters: repository.Parameters{}}
	}
	ref, err := repository.NewSnapshotRepository(tx).CreateGroupWithJobs(repository.NewGroup{
		ParcelID: parcelID, OwnerID: owner,
		DateStart: day("2025-01-01"), DateEnd: day("2025-04-30"),
		Members: members,
	})
	if err != nil {
		t.Fatal(err)
	}
	return ref
}

func TestGroupSealsWhenEveryMemberFinishes(t *testing.T) {
	tx := begin(t)
	owner := newUser(t, tx, "asset_holder")
	parcel := newParcel(t, tx, owner, "Group parcel")
	repo := repository.NewSnapshotRepository(tx)

	ref := newGroup(t, tx, owner, parcel.ID, "ndvi-s2-median", "chirps-rain-season")
	if ref.Members != 2 || ref.ShortID == "" {
		t.Fatalf("group ref = %+v", ref)
	}

	groups, err := repo.ListGroupsByParcelForOwner(parcel.ID, owner)
	if err != nil || len(groups) != 1 || len(groups[0].Members) != 2 {
		t.Fatalf("groups = %+v, %v", groups, err)
	}
	if g := groups[0]; g.DateStart != "2025-01-01" || g.RecordHash != nil || g.Members[0].Methodology != "chirps-rain-season" {
		t.Errorf("unsealed group = %+v", g)
	}

	sealed := func() *string {
		var hash *string
		if err := tx.Raw(`SELECT record_hash FROM snapshot_groups WHERE id = ?`, ref.ID).Scan(&hash).Error; err != nil {
			t.Fatal(err)
		}
		return hash
	}

	first := claim(t, tx, worker)
	complete(t, tx, first, clearResult)
	if sealed() != nil {
		t.Fatal("group sealed with a member still queued")
	}

	// A failed member still finishes the group; its record says so.
	second := claim(t, tx, worker)
	if err := tx.Exec(`SELECT fail_snapshot_job(?, ?, 'NO_IMAGERY', 'none', false)`, second.JobID, worker).Error; err != nil {
		t.Fatal(err)
	}
	hash := sealed()
	if hash == nil {
		t.Fatal("group not sealed after its last member finished")
	}

	var row struct {
		Recomputed string
		Text       string
		Problems   int
	}
	if err := tx.Raw(`
		SELECT group_record_hash(id, sealed_at) AS recomputed, group_record_text(id, sealed_at) AS text,
		       (SELECT count(*) FROM verify_snapshot_group(id)) AS problems
		FROM snapshot_groups WHERE id = ?`, ref.ID).Scan(&row).Error; err != nil {
		t.Fatal(err)
	}
	if row.Recomputed != *hash || row.Problems != 0 {
		t.Errorf("recomputed %s, stored %s, %d problems", row.Recomputed, *hash, row.Problems)
	}
	for _, want := range []string{"groundwork-group-v1", "date_start=2025-01-01", "member=chirps-rain-season@1.0.0:", "member=ndvi-s2-median@1.0.0:"} {
		if !strings.Contains(row.Text, want) {
			t.Errorf("group record text has no %q:\n%s", want, row.Text)
		}
	}
	if strings.Count(row.Text, ":failed") != 1 {
		t.Errorf("want exactly one failed member:\n%s", row.Text)
	}

	snapshots, err := repo.ListByParcelForOwner(parcel.ID, owner)
	if err != nil || len(snapshots) != 2 || snapshots[0].GroupShortID == nil || *snapshots[0].GroupShortID != ref.ShortID {
		t.Errorf("snapshots don't show their group: %+v, %v", snapshots, err)
	}

	expectError(t, tx, "Sealed groups cannot be changed", func(tx *gorm.DB) error {
		return tx.Exec(`UPDATE snapshot_groups SET date_end = '2025-05-31' WHERE id = ?`, ref.ID).Error
	})
	expectError(t, tx, "join before the group is sealed", func(tx *gorm.DB) error {
		return tx.Exec(`
			INSERT INTO snapshots (parcel_id, methodology_id, methodology_version, date_start, date_end,
			                       parameters, group_id, parcel_geometry, geometry_hash)
			SELECT g.parcel_id, m.id, m.version, g.date_start, g.date_end, '{}'::jsonb, g.id, p.geometry, 'x'
			FROM snapshot_groups g JOIN parcels p ON p.id = g.parcel_id
			JOIN methodologies m ON m.slug = 'ndvi-s2-median'
			WHERE g.id = ?`, ref.ID).Error
	})
}

func TestGroupMembersShareTheGroupsWindowAndAreUnique(t *testing.T) {
	tx := begin(t)
	owner := newUser(t, tx, "asset_holder")
	parcel := newParcel(t, tx, owner, "Group rules parcel")
	ref := newGroup(t, tx, owner, parcel.ID, "ndvi-s2-median", "chirps-rain-season")

	expectError(t, tx, "must have its group's parcel and dates", func(tx *gorm.DB) error {
		return tx.Exec(`
			INSERT INTO snapshots (parcel_id, methodology_id, methodology_version, date_start, date_end,
			                       parameters, group_id, parcel_geometry, geometry_hash)
			SELECT g.parcel_id, m.id, m.version, '2025-02-01', g.date_end, '{}'::jsonb, g.id, p.geometry, 'x'
			FROM snapshot_groups g JOIN parcels p ON p.id = g.parcel_id
			JOIN methodologies m ON m.slug = 'chirps-rain-season'
			WHERE g.id = ?`, ref.ID).Error
	})
	expectError(t, tx, "snapshots_group_methodology_idx", func(tx *gorm.DB) error {
		return tx.Exec(`
			INSERT INTO snapshots (parcel_id, methodology_id, methodology_version, date_start, date_end,
			                       parameters, group_id, parcel_geometry, geometry_hash)
			SELECT g.parcel_id, m.id, m.version, g.date_start, g.date_end, '{}'::jsonb, g.id, p.geometry, 'x'
			FROM snapshot_groups g JOIN parcels p ON p.id = g.parcel_id
			JOIN methodologies m ON m.slug = 'chirps-rain-season'
			WHERE g.id = ?`, ref.ID).Error
	})
	expectError(t, tx, "Snapshot input and audit fields are immutable", func(tx *gorm.DB) error {
		return tx.Exec(`UPDATE snapshots SET group_id = NULL WHERE group_id = ?`, ref.ID).Error
	})

	other := newUser(t, tx, "asset_holder")
	if _, err := repository.NewSnapshotRepository(tx).ListGroupsByParcelForOwner(parcel.ID, other); err != repository.ErrParcelNotFound {
		t.Errorf("another user listing the groups: %v, want ErrParcelNotFound", err)
	}
}
