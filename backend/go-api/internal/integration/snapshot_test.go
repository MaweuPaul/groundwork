package integration

import (
	"fmt"
	"strings"
	"testing"

	"gorm.io/gorm"

	"groundwork/internal/repository"
)

func TestSnapshotLifecycleSealsRecordV4(t *testing.T) {
	tx := begin(t)
	owner := newUser(t, tx, "asset_holder")
	parcel := newParcel(t, tx, owner, "Lifecycle parcel")

	id := queueSnapshot(t, tx, owner, parcel, repository.Parameters{"min_clear_pct": 40})
	job := claim(t, tx, worker)
	if job == nil || job.SnapshotID != id {
		t.Fatalf("claimed %+v, want snapshot %s", job, id)
	}
	if job.Attempt != 1 {
		t.Errorf("attempt = %d, want 1", job.Attempt)
	}
	want := repository.Parameters{"max_scene_cloud_pct": 20, "cloud_score_threshold": 0.6, "min_clear_pct": 40}
	for name, value := range want {
		if job.Parameters[name] != value {
			t.Errorf("claimed parameter %s = %v, want %v", name, job.Parameters[name], value)
		}
	}

	hash := complete(t, tx, job, result{
		Measurements: clearResult.Measurements,
		Sources:      clearResult.Sources,
		Flags:        map[string]string{"low_coverage": "Only 35% of the parcel was clear."},
	})
	if len(hash) != 64 {
		t.Fatalf("record hash %q is not a SHA-256 hex digest", hash)
	}

	var row struct {
		Status         string
		RecordFormat   int
		ChainSeq       int64
		RecordHash     string
		PrevRecordHash string
		Recomputed     string
		Text           string
	}
	err := tx.Raw(`
		SELECT status, record_format, chain_seq, record_hash, prev_record_hash,
		       snapshot_record_hash(id, completed_at, prev_record_hash) AS recomputed,
		       snapshot_record_text(id, completed_at, prev_record_hash) AS text
		FROM snapshots WHERE id = ?`, id).Scan(&row).Error
	if err != nil {
		t.Fatal(err)
	}
	if row.Status != "completed" || row.RecordFormat != currentFormat {
		t.Fatalf("status %s, format %d; want completed, %d", row.Status, row.RecordFormat, currentFormat)
	}
	if row.RecordHash != hash || row.Recomputed != hash {
		t.Errorf("stored %s, recomputed %s, returned %s: all three must match", row.RecordHash, row.Recomputed, hash)
	}
	for _, line := range []string{
		fmt.Sprintf("groundwork-snapshot-v%d", currentFormat),
		"code_version=" + codeVersion,
		"param=cloud_score_threshold:0.600000000",
		"param=max_scene_cloud_pct:20.000000000",
		"param=min_clear_pct:40.000000000",
		"measurement=ndvi_mean:0.612345678",
		"source=" + clearResult.Sources[0] + "@v1",
		"flag=low_coverage",
		"prev=" + row.PrevRecordHash,
	} {
		if !strings.Contains(row.Text, line+"\n") && !strings.HasSuffix(row.Text, line) {
			t.Errorf("record text is missing %q:\n%s", line, row.Text)
		}
	}
	if problems := chainProblems(t, tx); len(problems) != 0 {
		t.Errorf("chain problems: %v", problems)
	}

	detail, err := repository.NewSnapshotRepository(tx).GetByIDForOwner(id, owner)
	if err != nil || detail == nil {
		t.Fatalf("get snapshot: %v", err)
	}
	if detail.Parameters["min_clear_pct"] != 40 || len(detail.Flags) != 1 || detail.Flags[0].Flag != "low_coverage" {
		t.Errorf("detail parameters %v, flags %+v", detail.Parameters, detail.Flags)
	}
	if src := detail.ImagerySources; len(src) == 1 && (src[0].SourceVersion == nil || *src[0].SourceVersion == "") {
		t.Errorf("source %s has no version", src[0].SceneID)
	}
	if detail.Measurements["valid_pixel_pct"] != 87.5 || len(detail.ImagerySources) != 1 {
		t.Errorf("detail measurements %v, sources %d", detail.Measurements, len(detail.ImagerySources))
	}
	if mt := detail.MeasurementTypes["valid_pixel_pct"]; mt.Name != "Valid pixel percentage" || mt.Unit == nil || *mt.Unit != "%" {
		t.Errorf("valid_pixel_pct type = %+v", mt)
	}
	if len(detail.MeasurementTypes) != len(detail.Measurements) {
		t.Errorf("%d measurement types for %d measurements", len(detail.MeasurementTypes), len(detail.Measurements))
	}

	other := newUser(t, tx, "asset_holder")
	if hidden, err := repository.NewSnapshotRepository(tx).GetByIDForOwner(id, other); err != nil || hidden != nil {
		t.Errorf("another user can read the snapshot: %+v, %v", hidden, err)
	}
}

func TestChainLinksEachRecordToThePrevious(t *testing.T) {
	tx := begin(t)
	owner := newUser(t, tx, "asset_holder")
	parcel := newParcel(t, tx, owner, "Chain parcel")

	first, firstHash := runSnapshot(t, tx, owner, parcel, clearResult)
	second, _ := runSnapshot(t, tx, owner, parcel, clearResult)

	var links []struct {
		ID             string
		ParcelSeq      int64
		PrevRecordHash string
	}
	if err := tx.Raw(`SELECT id, parcel_seq, prev_record_hash FROM snapshots WHERE id IN (?, ?) ORDER BY parcel_seq`,
		first, second).Scan(&links).Error; err != nil {
		t.Fatal(err)
	}
	if len(links) != 2 || links[0].ParcelSeq != 1 || links[1].ParcelSeq != 2 {
		t.Fatalf("parcel_seq should be 1 then 2: %+v", links)
	}
	if links[0].PrevRecordHash != genesis {
		t.Errorf("the parcel's first record links to %s, want the genesis hash", links[0].PrevRecordHash)
	}
	if links[1].PrevRecordHash != firstHash {
		t.Errorf("second record's prev is %s, want the first record's hash %s", links[1].PrevRecordHash, firstHash)
	}
	if problems := chainProblems(t, tx); len(problems) != 0 {
		t.Errorf("chain problems: %v", problems)
	}
}

func TestCompletedSnapshotsAreFrozen(t *testing.T) {
	tx := begin(t)
	owner := newUser(t, tx, "asset_holder")
	parcel := newParcel(t, tx, owner, "Frozen parcel")
	id, _ := runSnapshot(t, tx, owner, parcel, clearResult)

	cases := []struct {
		name string
		want string
		sql  string
	}{
		{"edit a field", "cannot be changed", `UPDATE snapshots SET failure_message = 'x' WHERE id = ?`},
		{"delete it", "cannot be deleted", `DELETE FROM snapshots WHERE id = ?`},
		{"edit a measurement", "cannot be changed", `UPDATE measurements SET value = 0.99 WHERE snapshot_id = ?`},
		{"remove a source", "cannot be changed", `DELETE FROM imagery_sources WHERE snapshot_id = ?`},
		{"add a flag", "cannot be added", `INSERT INTO snapshot_flags (snapshot_id, flag) VALUES (?, 'late_flag')`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			expectError(t, tx, c.want, func(tx *gorm.DB) error { return tx.Exec(c.sql, id).Error })
		})
	}

	if problems := chainProblems(t, tx); len(problems) != 0 {
		t.Errorf("chain problems after refused edits: %v", problems)
	}
}

func TestTamperingIsDetectedByVerify(t *testing.T) {
	tx := begin(t)
	owner := newUser(t, tx, "asset_holder")
	parcel := newParcel(t, tx, owner, "Tamper parcel")
	id, _ := runSnapshot(t, tx, owner, parcel, clearResult)

	// Someone with enough access to switch the triggers off edits a sealed measurement.
	for _, sql := range []string{
		`ALTER TABLE measurements DISABLE TRIGGER USER`,
		`UPDATE measurements SET value = 0.95 WHERE snapshot_id = '` + id.String() + `'`,
		`ALTER TABLE measurements ENABLE TRIGGER USER`,
	} {
		if err := tx.Exec(sql).Error; err != nil {
			t.Fatal(err)
		}
	}

	problems := chainProblems(t, tx)
	if len(problems) != 1 || !strings.Contains(problems[0], "does not match the stored contents") {
		t.Errorf("verify_snapshot_chain() = %v, want one content mismatch", problems)
	}
}

func TestSnapshotNeedsMeasurementsAndSources(t *testing.T) {
	tx := begin(t)
	owner := newUser(t, tx, "asset_holder")
	parcel := newParcel(t, tx, owner, "Empty result parcel")
	queueSnapshot(t, tx, owner, parcel, nil)
	job := claim(t, tx, worker)

	expectError(t, tx, "without measurements", func(tx *gorm.DB) error {
		return tx.Exec(`SELECT complete_snapshot_job(?, ?)`, job.JobID, worker).Error
	})
	expectError(t, tx, "without imagery sources", func(tx *gorm.DB) error {
		save(t, tx, job.SnapshotID, result{Measurements: clearResult.Measurements})
		return tx.Exec(`SELECT complete_snapshot_job(?, ?)`, job.JobID, worker).Error
	})
}
