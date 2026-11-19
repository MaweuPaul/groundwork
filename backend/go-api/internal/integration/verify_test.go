package integration

import (
	"strings"
	"testing"

	"groundwork/internal/repository"
)

func TestVerifyParcelReportsTampering(t *testing.T) {
	tx := begin(t)
	owner := newUser(t, tx, "asset_holder")
	parcel := newParcel(t, tx, owner, "Verify parcel")
	repo := repository.NewSnapshotRepository(tx)

	empty, err := repo.VerifyParcelForOwner(parcel.ID, owner)
	if err != nil || !empty.Intact || empty.Records != 0 || empty.HeadSeq != nil {
		t.Fatalf("empty parcel = %+v, %v", empty, err)
	}

	first, _ := runSnapshot(t, tx, owner, parcel, clearResult)
	_, head := runSnapshot(t, tx, owner, parcel, clearResult)
	group := newGroup(t, tx, owner, parcel.ID, "ndvi-s2-median", "chirps-rain-season")
	complete(t, tx, claim(t, tx, worker), clearResult)
	complete(t, tx, claim(t, tx, worker), clearResult)

	check, err := repo.VerifyParcelForOwner(parcel.ID, owner)
	if err != nil {
		t.Fatal(err)
	}
	if !check.Intact || check.Records != 4 || check.Groups != 1 || check.HeadSeq == nil || *check.HeadSeq != 4 {
		t.Errorf("intact parcel = %+v", check)
	}
	if check.CheckedAt.IsZero() || len(check.Problems) != 0 {
		t.Errorf("intact parcel = %+v", check)
	}
	if head == "" || check.HeadHash == nil {
		t.Errorf("head hash = %v", check.HeadHash)
	}

	inSavepoint(t, tx, "tamper_record", func() {
		withoutTriggers(t, tx, "measurements", `UPDATE measurements SET value = 0.95 WHERE snapshot_id = ?`, first)
		check, err := repo.VerifyParcelForOwner(parcel.ID, owner)
		if err != nil {
			t.Fatal(err)
		}
		if check.Intact || len(check.Problems) != 1 {
			t.Fatalf("tampered record = %+v", check)
		}
		p := check.Problems[0]
		if p.SnapshotID == nil || *p.SnapshotID != first || p.ShortID == nil || p.Seq == nil || *p.Seq != 1 ||
			!strings.Contains(p.Problem, "does not match the stored contents") {
			t.Errorf("problem = %+v", p)
		}
	})

	inSavepoint(t, tx, "tamper_group", func() {
		withoutTriggers(t, tx, "snapshot_groups", `UPDATE snapshot_groups SET record_hash = repeat('f', 64) WHERE id = ?`, group.ID)
		check, err := repo.VerifyParcelForOwner(parcel.ID, owner)
		if err != nil {
			t.Fatal(err)
		}
		if check.Intact || len(check.Problems) != 1 {
			t.Fatalf("tampered group = %+v", check)
		}
		if p := check.Problems[0]; p.GroupShortID == nil || *p.GroupShortID != group.ShortID || p.SnapshotID != nil {
			t.Errorf("problem = %+v", p)
		}
	})

	other := newUser(t, tx, "asset_holder")
	if _, err := repo.VerifyParcelForOwner(parcel.ID, other); err != repository.ErrParcelNotFound {
		t.Errorf("another user verifying the parcel: %v, want ErrParcelNotFound", err)
	}
}
