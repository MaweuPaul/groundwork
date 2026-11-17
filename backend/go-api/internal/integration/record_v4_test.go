package integration

import (
	"fmt"
	"strings"
	"testing"

	"gorm.io/gorm"

	"groundwork/internal/repository"
)

func recordText(t *testing.T, tx *gorm.DB, snapshotID any) string {
	t.Helper()
	var text string
	if err := tx.Raw(`
		SELECT snapshot_record_text(id, completed_at, prev_record_hash) FROM snapshots WHERE id = ?`,
		snapshotID,
	).Scan(&text).Error; err != nil {
		t.Fatal(err)
	}
	return text
}

func TestRecordSealsManifestAndCodeVersion(t *testing.T) {
	tx := begin(t)
	owner := newUser(t, tx, "asset_holder")
	parcel := newParcel(t, tx, owner, "Sealed code")
	id, hash := runSnapshot(t, tx, owner, parcel, clearResult)

	var manifestHash string
	if err := tx.Raw(`SELECT methodology_manifest_hash(?)`, ndvi(t, tx).ID).Scan(&manifestHash).Error; err != nil {
		t.Fatal(err)
	}
	if len(manifestHash) != 64 {
		t.Fatalf("manifest hash %q, want 64 hex characters", manifestHash)
	}

	text := recordText(t, tx, id)
	lines := strings.Split(text, "\n")
	if want := fmt.Sprintf("groundwork-snapshot-v%d", currentFormat); lines[0] != want {
		t.Fatalf("record starts with %q, want %q", lines[0], want)
	}
	for _, want := range []string{"manifest_sha256=" + manifestHash, "code_version=" + codeVersion} {
		if !strings.Contains(text, "\n"+want+"\n") {
			t.Fatalf("record lacks %q:\n%s", want, text)
		}
	}
	if problems := chainProblems(t, tx); len(problems) > 0 {
		t.Fatalf("chain problems: %v", problems)
	}
	detail, err := repository.NewSnapshotRepository(tx).GetByIDForOwner(id, owner)
	if err != nil || detail == nil || detail.CodeVersion == nil || *detail.CodeVersion != codeVersion ||
		detail.ManifestSHA256 == nil || *detail.ManifestSHA256 != manifestHash {
		t.Fatalf("detail doesn't show what was sealed: %+v, %v", detail, err)
	}
	if hash == "" {
		t.Fatal("no record hash returned")
	}

	// The version is inside the hash: changing it behind the triggers' back breaks the chain.
	if err := tx.Exec(`ALTER TABLE snapshots DISABLE TRIGGER USER`).Error; err != nil {
		t.Fatal(err)
	}
	if err := tx.Exec(`UPDATE snapshots SET code_version = 'tampered' WHERE id = ?`, id).Error; err != nil {
		t.Fatal(err)
	}
	if err := tx.Exec(`ALTER TABLE snapshots ENABLE TRIGGER USER`).Error; err != nil {
		t.Fatal(err)
	}
	if problems := chainProblems(t, tx); len(problems) == 0 {
		t.Fatal("a changed code version went unnoticed")
	}
}

func TestV4RecordNeedsCodeVersion(t *testing.T) {
	tx := begin(t)
	owner := newUser(t, tx, "asset_holder")
	parcel := newParcel(t, tx, owner, "No version")
	queueSnapshot(t, tx, owner, parcel, nil)
	job := claim(t, tx, worker)
	save(t, tx, job.SnapshotID, clearResult)

	expectError(t, tx, "snapshots_code_version_sealed", func(tx *gorm.DB) error {
		return tx.Exec(`SELECT complete_snapshot_job(?, ?)`, job.JobID, worker).Error
	})
	expectError(t, tx, "snapshots_code_version_sealed", func(tx *gorm.DB) error {
		return tx.Exec(`SELECT complete_snapshot_job(?, ?, '  ')`, job.JobID, worker).Error
	})
}

func TestOlderRecordsKeepTheirText(t *testing.T) {
	tx := begin(t)
	owner := newUser(t, tx, "asset_holder")
	parcel := newParcel(t, tx, owner, "Format 3")
	id, _ := runSnapshot(t, tx, owner, parcel, clearResult)

	// Re-label the record as v3 and reseal it the way migration 013 would have.
	if err := tx.Exec(`ALTER TABLE snapshots DISABLE TRIGGER USER`).Error; err != nil {
		t.Fatal(err)
	}
	if err := tx.Exec(`
		UPDATE snapshots SET record_format = 3, code_version = NULL WHERE id = ?;`, id).Error; err != nil {
		t.Fatal(err)
	}
	if err := tx.Exec(`
		UPDATE snapshots SET record_hash = snapshot_record_hash(id, completed_at, prev_record_hash) WHERE id = ?`, id).Error; err != nil {
		t.Fatal(err)
	}
	if err := tx.Exec(`ALTER TABLE snapshots ENABLE TRIGGER USER`).Error; err != nil {
		t.Fatal(err)
	}

	text := recordText(t, tx, id)
	if strings.Contains(text, "manifest_sha256=") || strings.Contains(text, "code_version=") {
		t.Fatalf("a v3 record grew v4 lines:\n%s", text)
	}
	if !strings.HasPrefix(text, "groundwork-snapshot-v3\nsnapshot_id=") {
		t.Fatalf("v3 text changed:\n%s", text)
	}
}
