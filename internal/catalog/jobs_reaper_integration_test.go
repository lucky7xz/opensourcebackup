//go:build integration

package catalog_test

import (
	"context"
	"testing"
	"time"

	"github.com/cerberus8484/opensourcebackup/internal/catalog"
)

// TestJobStore_FailStaleJobs verifies the Stale-Job-Reaper logic: running jobs
// that have gone silent are auto-failed, while live and terminal jobs are left
// untouched. Timestamps are set via raw SQL for deterministic control.
func TestJobStore_FailStaleJobs(t *testing.T) {
	db := newTestDB(t)
	truncateTables(t, db)
	sys, pol := createFixtureSystemAndPolicy(t, db)
	store := catalog.NewJobStore(db)
	ctx := context.Background()

	// Helper: create a running job, then force its timestamps via raw SQL.
	mkJob := func(status string, startedAgo time.Duration, lastProgressAgo *time.Duration) *catalog.BackupJob {
		j := &catalog.BackupJob{SystemID: sys.ID, PolicyID: pol.ID, Status: status}
		if err := store.Create(ctx, j); err != nil {
			t.Fatalf("create job: %v", err)
		}
		started := time.Now().Add(-startedAgo)
		if lastProgressAgo == nil {
			_, err := db.Pool().Exec(ctx,
				`UPDATE backup_jobs SET started_at=$1, last_progress_at=NULL WHERE id=$2`,
				started, j.ID)
			if err != nil {
				t.Fatalf("set timestamps: %v", err)
			}
		} else {
			lp := time.Now().Add(-*lastProgressAgo)
			_, err := db.Pool().Exec(ctx,
				`UPDATE backup_jobs SET started_at=$1, last_progress_at=$2 WHERE id=$3`,
				started, lp, j.ID)
			if err != nil {
				t.Fatalf("set timestamps: %v", err)
			}
		}
		return j
	}

	dur := func(d time.Duration) *time.Duration { return &d }

	// A: running, started 13h ago, never reported progress → STALE (start grace 12h)
	jobA := mkJob("running", 13*time.Hour, nil)
	// B: running, started 1h ago, no progress yet → live (within start grace)
	jobB := mkJob("running", 1*time.Hour, nil)
	// C: running 13h, but reported progress 1m ago → live (recent heartbeat)
	jobC := mkJob("running", 13*time.Hour, dur(1*time.Minute))
	// D: running, reported progress 40m ago → STALE (progress grace 30m)
	jobD := mkJob("running", 2*time.Hour, dur(40*time.Minute))
	// E: success, started 13h ago → never reaped (terminal status)
	jobE := mkJob("success", 13*time.Hour, nil)

	n, err := store.FailStaleJobs(ctx, 30*time.Minute, 12*time.Hour)
	if err != nil {
		t.Fatalf("FailStaleJobs: %v", err)
	}
	if n != 2 {
		t.Errorf("want 2 reaped, got %d", n)
	}

	wantStatus := func(j *catalog.BackupJob, want string) {
		got, err := store.GetByID(ctx, j.ID)
		if err != nil {
			t.Fatalf("get %s: %v", j.ID, err)
		}
		if got.Status != want {
			t.Errorf("job status: want %q, got %q", want, got.Status)
		}
		if want == "failed" {
			if got.FinishedAt == nil {
				t.Error("reaped job must have finished_at set")
			}
			if got.ErrorSummary == nil || *got.ErrorSummary == "" {
				t.Error("reaped job must have an error_summary")
			}
		}
	}

	wantStatus(jobA, "failed")  // stale by start grace
	wantStatus(jobB, "running") // still live
	wantStatus(jobC, "running") // recent heartbeat
	wantStatus(jobD, "failed")  // stale by progress grace
	wantStatus(jobE, "success") // terminal, untouched
}
