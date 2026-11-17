package integration

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// genesis is the previous hash of a parcel's first record.
var genesis = strings.Repeat("0", 64)

func parcelChainProblems(t *testing.T, tx *gorm.DB, parcelID uuid.UUID) []string {
	t.Helper()
	var problems []string
	if err := tx.Raw(`SELECT problem FROM verify_parcel_chain(?)`, parcelID).Scan(&problems).Error; err != nil {
		t.Fatalf("verify parcel chain: %v", err)
	}
	return problems
}

// withoutTriggers runs sql with table's own triggers off: a stand-in for someone editing the
// database behind the application's back.
func withoutTriggers(t *testing.T, tx *gorm.DB, table, sql string, args ...any) {
	t.Helper()
	if err := tx.Exec(`ALTER TABLE ` + table + ` DISABLE TRIGGER USER`).Error; err != nil {
		t.Fatal(err)
	}
	if err := tx.Exec(sql, args...).Error; err != nil {
		t.Fatal(err)
	}
	if err := tx.Exec(`ALTER TABLE ` + table + ` ENABLE TRIGGER USER`).Error; err != nil {
		t.Fatal(err)
	}
}

// inSavepoint runs fn and undoes whatever it changed.
func inSavepoint(t *testing.T, tx *gorm.DB, name string, fn func()) {
	t.Helper()
	if err := tx.SavePoint(name).Error; err != nil {
		t.Fatal(err)
	}
	defer tx.RollbackTo(name)
	fn()
}

func TestEachParcelHasItsOwnChain(t *testing.T) {
	tx := begin(t)
	owner := newUser(t, tx, "asset_holder")
	a := newParcel(t, tx, owner, "Parcel A")
	b := newParcel(t, tx, owner, "Parcel B")

	_, a1 := runSnapshot(t, tx, owner, a, clearResult)
	b1ID, _ := runSnapshot(t, tx, owner, b, clearResult)
	a2ID, _ := runSnapshot(t, tx, owner, a, clearResult)

	var rows []struct {
		ID             uuid.UUID
		ParcelSeq      int64
		PrevRecordHash string
	}
	if err := tx.Raw(`SELECT id, parcel_seq, prev_record_hash FROM snapshots WHERE id IN (?, ?)`, b1ID, a2ID).
		Scan(&rows).Error; err != nil {
		t.Fatal(err)
	}
	for _, r := range rows {
		switch r.ID {
		case b1ID:
			if r.ParcelSeq != 1 || r.PrevRecordHash != genesis {
				t.Errorf("B's first record: seq %d, prev %s; want 1 and genesis, not A's record", r.ParcelSeq, r.PrevRecordHash)
			}
		case a2ID:
			if r.ParcelSeq != 2 || r.PrevRecordHash != a1 {
				t.Errorf("A's second record: seq %d, prev %s; want 2 and A's first record %s", r.ParcelSeq, r.PrevRecordHash, a1)
			}
		}
	}

	if p := parcelChainProblems(t, tx, a.ID); len(p) > 0 {
		t.Errorf("parcel A: %v", p)
	}
	if p := chainProblems(t, tx); len(p) > 0 {
		t.Errorf("all chains: %v", p)
	}
}

func TestParcelChainCatchesTampering(t *testing.T) {
	tx := begin(t)
	owner := newUser(t, tx, "asset_holder")
	parcel := newParcel(t, tx, owner, "Tampered")
	other := newParcel(t, tx, owner, "Untouched")
	runSnapshot(t, tx, owner, parcel, clearResult)
	_, second := runSnapshot(t, tx, owner, parcel, clearResult)
	runSnapshot(t, tx, owner, other, clearResult)

	secondID := `(SELECT id FROM snapshots WHERE record_hash = ?)`

	inSavepoint(t, tx, "changed_contents", func() {
		withoutTriggers(t, tx, "measurements", `UPDATE measurements SET value = 0.1 WHERE snapshot_id = `+secondID, second)
		if p := parcelChainProblems(t, tx, parcel.ID); len(p) == 0 {
			t.Error("a changed measurement went unnoticed")
		}
		if p := parcelChainProblems(t, tx, other.ID); len(p) > 0 {
			t.Errorf("another parcel's chain was affected: %v", p)
		}
	})

	inSavepoint(t, tx, "last_record_removed", func() {
		for _, table := range []string{"measurements", "imagery_sources", "snapshot_flags", "snapshot_jobs"} {
			withoutTriggers(t, tx, table, `DELETE FROM `+table+` WHERE snapshot_id = `+secondID, second)
		}
		withoutTriggers(t, tx, "snapshots", `DELETE FROM snapshots WHERE record_hash = ?`, second)
		p := parcelChainProblems(t, tx, parcel.ID)
		if len(p) == 0 || !strings.Contains(strings.Join(p, " "), "chain head") {
			t.Errorf("a removed last record went unnoticed: %v", p)
		}
	})
}
