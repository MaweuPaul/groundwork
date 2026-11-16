package integration

import (
	"testing"

	"groundwork/internal/repository"
)

func TestActivityShowsOnlyTheOwnersEvents(t *testing.T) {
	tx := begin(t)
	owner := newUser(t, tx, "asset_holder")
	other := newUser(t, tx, "asset_holder")
	parcel := newParcel(t, tx, owner, "Activity parcel")
	newParcel(t, tx, other, "Someone else's parcel")

	runSnapshot(t, tx, owner, parcel, result{
		Measurements: clearResult.Measurements,
		Sources:      clearResult.Sources,
		Flags:        map[string]string{"low_coverage": "Only part was clear."},
	})
	queueSnapshot(t, tx, owner, parcel, nil) // still pending

	repo := repository.NewActivityRepository(tx)
	events, err := repo.ListForOwner(owner, 50)
	if err != nil {
		t.Fatal(err)
	}

	kinds := map[string]int{}
	for _, e := range events {
		kinds[e.Kind]++
		if e.ParcelID != parcel.ID {
			t.Errorf("event %s is for parcel %s, not the owner's", e.Kind, e.ParcelID)
		}
		if e.Kind == repository.ActivitySnapshotCompleted {
			if e.NdviMean == nil || *e.NdviMean != clearResult.Measurements["ndvi_mean"] {
				t.Errorf("completed event NDVI = %v", e.NdviMean)
			}
			if e.RecordHash == nil || len(e.Flags) != 1 || e.Flags[0] != "low_coverage" {
				t.Errorf("completed event hash %v, flags %v", e.RecordHash, e.Flags)
			}
		}
	}
	// The completed snapshot's queued event is hidden; only the pending one shows.
	want := map[string]int{
		repository.ActivityParcelRegistered:  1,
		repository.ActivitySnapshotCompleted: 1,
		repository.ActivitySnapshotQueued:    1,
	}
	for kind, n := range want {
		if kinds[kind] != n {
			t.Errorf("%d %s events, want %d (all: %v)", kinds[kind], kind, n, kinds)
		}
	}
	limited, err := repo.ListForOwner(owner, 1)
	if err != nil || len(limited) != 1 {
		t.Errorf("limit 1 returned %d events, err %v", len(limited), err)
	}

	theirs, err := repo.ListForOwner(other, 50)
	if err != nil || len(theirs) != 1 || theirs[0].Kind != repository.ActivityParcelRegistered {
		t.Errorf("other user's activity = %+v, err %v", theirs, err)
	}
}
