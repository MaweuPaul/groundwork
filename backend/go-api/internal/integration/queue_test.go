package integration

import (
	"testing"

	"gorm.io/gorm"
)

func TestRetryableFailureRequeuesWithBackoff(t *testing.T) {
	tx := begin(t)
	owner := newUser(t, tx, "asset_holder")
	parcel := newParcel(t, tx, owner, "Retry parcel")
	id := queueSnapshot(t, tx, owner, parcel, nil)
	job := claim(t, tx, worker)

	var status string
	if err := tx.Raw(`SELECT fail_snapshot_job(?, ?, 'EE_TIMEOUT', 'Earth Engine timed out.', true)::text`,
		job.JobID, worker).Scan(&status).Error; err != nil {
		t.Fatal(err)
	}
	if status != "queued" {
		t.Fatalf("status = %s, want queued", status)
	}

	var row struct {
		JobStatus      string
		SnapshotStatus string
		Waiting        bool
		LastErrorCode  string
	}
	if err := tx.Raw(`
		SELECT j.status AS job_status, s.status AS snapshot_status, j.run_after > NOW() AS waiting, j.last_error_code
		FROM snapshot_jobs j JOIN snapshots s ON s.id = j.snapshot_id
		WHERE s.id = ?`, id).Scan(&row).Error; err != nil {
		t.Fatal(err)
	}
	if row.JobStatus != "queued" || row.SnapshotStatus == "failed" || !row.Waiting || row.LastErrorCode != "EE_TIMEOUT" {
		t.Errorf("after a retryable failure: %+v", row)
	}
	if again := claim(t, tx, worker); again != nil {
		t.Errorf("claimed job %s again before its backoff ended", again.JobID)
	}
}

func TestPermanentFailureFailsTheSnapshot(t *testing.T) {
	tx := begin(t)
	owner := newUser(t, tx, "asset_holder")
	parcel := newParcel(t, tx, owner, "Failing parcel")
	id := queueSnapshot(t, tx, owner, parcel, nil)
	job := claim(t, tx, worker)

	var status string
	if err := tx.Raw(`SELECT fail_snapshot_job(?, ?, 'NO_IMAGERY', 'No scenes passed the cloud limit.', false)::text`,
		job.JobID, worker).Scan(&status).Error; err != nil {
		t.Fatal(err)
	}
	if status != "failed" {
		t.Fatalf("status = %s, want failed", status)
	}

	var row struct {
		Status         string
		FailureMessage string
		RecordHash     *string
	}
	if err := tx.Raw(`SELECT status, failure_message, record_hash FROM snapshots WHERE id = ?`, id).Scan(&row).Error; err != nil {
		t.Fatal(err)
	}
	if row.Status != "failed" || row.FailureMessage != "NO_IMAGERY: No scenes passed the cloud limit." || row.RecordHash != nil {
		t.Errorf("failed snapshot: %+v", row)
	}
	if problems := chainProblems(t, tx); len(problems) != 0 {
		t.Errorf("a failed snapshot broke the chain: %v", problems)
	}
}

func TestOnlyTheLockHolderCanFinishAJob(t *testing.T) {
	tx := begin(t)
	owner := newUser(t, tx, "asset_holder")
	parcel := newParcel(t, tx, owner, "Locked parcel")
	queueSnapshot(t, tx, owner, parcel, nil)
	job := claim(t, tx, worker)

	if other := claim(t, tx, "second-worker"); other != nil {
		t.Fatalf("a second worker claimed locked job %s", other.JobID)
	}
	expectError(t, tx, "is not held by worker", func(tx *gorm.DB) error {
		save(t, tx, job.SnapshotID, clearResult)
		return tx.Exec(`SELECT complete_snapshot_job(?, 'second-worker')`, job.JobID).Error
	})
	expectError(t, tx, "is not held by worker", func(tx *gorm.DB) error {
		return tx.Exec(`SELECT fail_snapshot_job(?, 'second-worker', 'X', 'x', false)`, job.JobID).Error
	})

	complete(t, tx, job, clearResult)
}
