package agent_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"log/slog"
	"os"

	"github.com/google/uuid"

	"github.com/cerberus8484/opensourcebackup/internal/agent"
	"github.com/cerberus8484/opensourcebackup/internal/catalog"
)

var testLog = slog.New(slog.NewTextHandler(os.Stderr, nil))

// stubCP is a controllable ControlPlaneClient for testing.
type stubCP struct {
	jobs        []catalog.BackupJob
	policy      *catalog.BackupPolicy
	updatedJobs []*catalog.BackupJob
	snapshots   []*catalog.Snapshot
	err         error
}

func (s *stubCP) ListPendingJobs(_ context.Context, _ uuid.UUID) ([]catalog.BackupJob, error) {
	return s.jobs, s.err
}

func (s *stubCP) GetPolicy(_ context.Context, _ uuid.UUID) (*catalog.BackupPolicy, error) {
	if s.policy == nil {
		return nil, errors.New("policy not found")
	}
	return s.policy, nil
}

func (s *stubCP) UpdateJobStatus(_ context.Context, j *catalog.BackupJob) error {
	cp := *j
	s.updatedJobs = append(s.updatedJobs, &cp)
	return nil
}

func (s *stubCP) CreateSnapshot(_ context.Context, snap *catalog.Snapshot) error {
	cp := *snap
	s.snapshots = append(s.snapshots, &cp)
	return nil
}

func TestAgent_FailsJob_WhenPolicyHasNoRepository(t *testing.T) {
	repoID := uuid.Nil // no repository
	policyID := uuid.New()
	systemID := uuid.New()

	cp := &stubCP{
		jobs: []catalog.BackupJob{{
			ID:       uuid.New(),
			SystemID: systemID,
			PolicyID: policyID,
			Status:   "pending",
		}},
		policy: &catalog.BackupPolicy{
			ID:           policyID,
			Name:         "no-repo-policy",
			Engine:       "restic",
			RepositoryID: nil, // no repository configured
		},
	}
	_ = repoID

	cfg := agent.Config{
		SystemID:       systemID,
		PollInterval:   time.Hour,
		ResticBin:      "restic-nonexistent",
		ResticPassword: "test",
		ResticRepo:     "s3:test",
	}
	a := agent.New(cfg, cp, testLog)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	a.Run(ctx) //nolint:errcheck // context timeout is expected

	// Job must be marked failed
	if len(cp.updatedJobs) == 0 {
		t.Fatal("expected job status update")
	}
	lastUpdate := cp.updatedJobs[len(cp.updatedJobs)-1]
	if lastUpdate.Status != "failed" {
		t.Errorf("want status=failed, got %s", lastUpdate.Status)
	}
	if lastUpdate.ErrorSummary == nil || *lastUpdate.ErrorSummary == "" {
		t.Error("expected error summary to be set")
	}
}

func TestAgent_SetsJobRunning_BeforeBackup(t *testing.T) {
	repoID := uuid.New()
	policyID := uuid.New()
	systemID := uuid.New()

	cp := &stubCP{
		jobs: []catalog.BackupJob{{
			ID:       uuid.New(),
			SystemID: systemID,
			PolicyID: policyID,
			Status:   "pending",
		}},
		policy: &catalog.BackupPolicy{
			ID:           policyID,
			Name:         "test-policy",
			Engine:       "restic",
			RepositoryID: &repoID,
			Includes:     []string{"/tmp"},
		},
	}

	cfg := agent.Config{
		SystemID:       systemID,
		PollInterval:   time.Hour,
		ResticBin:      "restic-nonexistent", // will fail at backup, but running is set first
		ResticPassword: "test",
		ResticRepo:     "s3:test",
	}
	a := agent.New(cfg, cp, testLog)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	a.Run(ctx) //nolint:errcheck

	// First update must be status=running
	if len(cp.updatedJobs) == 0 {
		t.Fatal("expected at least one job update")
	}
	if cp.updatedJobs[0].Status != "running" {
		t.Errorf("first update: want running, got %s", cp.updatedJobs[0].Status)
	}
}
