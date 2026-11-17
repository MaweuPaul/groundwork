package integration

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"

	"gorm.io/gorm"

	"groundwork/internal/repository"
)

var mapImage = artifact{
	Key:       "ndvi_map",
	MediaType: "image/png",
	Content:   []byte("\x89PNG fake image bytes"),
	Metadata:  `{"width": 512, "height": 256, "bounds": [36.9, -0.4, 37.0, -0.3], "visualization": {"min": 0, "max": 0.9}}`,
}

func TestArtifactsAreSealedIntoTheRecord(t *testing.T) {
	tx := begin(t)
	owner := newUser(t, tx, "asset_holder")
	parcel := newParcel(t, tx, owner, "With map")
	r := clearResult
	r.Artifacts = []artifact{mapImage}
	id, _ := runSnapshot(t, tx, owner, parcel, r)

	sum := sha256.Sum256(mapImage.Content)
	contentHash := hex.EncodeToString(sum[:])
	text := recordText(t, tx, id)
	if !strings.Contains(text, "\nartifact=ndvi_map:image/png:"+contentHash+":") {
		t.Fatalf("record doesn't seal the image's hash %s:\n%s", contentHash, text)
	}
	if p := chainProblems(t, tx); len(p) > 0 {
		t.Fatalf("chain problems: %v", p)
	}

	repo := repository.NewSnapshotRepository(tx)
	detail, err := repo.GetByIDForOwner(id, owner)
	if err != nil || detail == nil || len(detail.Artifacts) != 1 {
		t.Fatalf("detail artifacts: %+v, %v", detail, err)
	}
	a := detail.Artifacts[0]
	if a.SHA256 != contentHash || !a.Sealed || a.ByteSize != len(mapImage.Content) {
		t.Errorf("artifact info %+v", a)
	}
	if detail.NdviImage == nil || detail.NdviImage.Width != 512 || detail.NdviImage.Bounds[3] != -0.3 {
		t.Errorf("NDVI image placement from metadata: %+v", detail.NdviImage)
	}

	got, err := repo.ArtifactForOwner(id, owner, "ndvi_map")
	if err != nil || got == nil || string(got.Content) != string(mapImage.Content) || got.MediaType != "image/png" {
		t.Fatalf("serving the artifact: %+v, %v", got, err)
	}
	stranger := newUser(t, tx, "asset_holder")
	if got, _ := repo.ArtifactForOwner(id, stranger, "ndvi_map"); got != nil {
		t.Error("another user can read the artifact")
	}

	// Frozen with the snapshot, like measurements.
	expectError(t, tx, "completed snapshot", func(tx *gorm.DB) error {
		return tx.Exec(`UPDATE artifacts SET content = 'other' WHERE snapshot_id = ?`, id).Error
	})
	expectError(t, tx, "completed snapshot", func(tx *gorm.DB) error {
		return tx.Exec(`INSERT INTO artifacts (snapshot_id, key, media_type, content) VALUES (?, 'late', 'text/csv', 'x')`, id).Error
	})

	// Swapping the image behind the triggers' back breaks the chain, and so does moving it.
	inSavepoint(t, tx, "swapped_image", func() {
		withoutTriggers(t, tx, "artifacts", `UPDATE artifacts SET content = 'other' WHERE snapshot_id = ?`, id)
		if p := parcelChainProblems(t, tx, parcel.ID); len(p) == 0 {
			t.Error("a swapped image went unnoticed")
		}
	})
	inSavepoint(t, tx, "moved_image", func() {
		withoutTriggers(t, tx, "artifacts", `UPDATE artifacts SET metadata = jsonb_set(metadata, '{bounds,0}', '10') WHERE snapshot_id = ?`, id)
		if p := parcelChainProblems(t, tx, parcel.ID); len(p) == 0 {
			t.Error("a changed image placement went unnoticed")
		}
	})
}

func TestSnapshotWithoutArtifactsStillSeals(t *testing.T) {
	tx := begin(t)
	owner := newUser(t, tx, "asset_holder")
	parcel := newParcel(t, tx, owner, "No map")
	r := clearResult
	r.Flags = map[string]string{"map_image_failed": "computePixels: too many pixels"}
	id, _ := runSnapshot(t, tx, owner, parcel, r)

	text := recordText(t, tx, id)
	if strings.Contains(text, "artifact=") || !strings.Contains(text, "\nflag=map_image_failed\n") {
		t.Fatalf("record:\n%s", text)
	}
	if p := chainProblems(t, tx); len(p) > 0 {
		t.Fatalf("chain problems: %v", p)
	}
}
