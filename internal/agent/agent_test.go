package agent_test

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"

	agentclient "github.com/cerberus8484/opensourcebackup/internal/agent/client"
	"github.com/cerberus8484/opensourcebackup/internal/catalog"

	"github.com/cerberus8484/opensourcebackup/internal/agent"
)

var testLog = slog.New(slog.NewTextHandler(os.Stderr, nil))

// stubCP implements agent.ControlPlaneClient for testing.
type stubCP struct {
	jobs      []catalog.BackupJob
	policy    *catalog.BackupPolicy
	started   []uuid.UUID
	completed []uuid.UUID
	failed    []failRecord
	listErr   error
}

type failRecord struct {
	jobID  uuid.UUID
	reason string
}

func (s *stubCP) ListPendingJobs(_ context.Context) ([]catalog.BackupJob, error) {
	return s.jobs, s.listErr
}

func (s *stubCP) GetPolicy(_ context.Context, _ uuid.UUID) (*catalog.BackupPolicy, error) {
	if s.policy == nil {
		return nil, errors.New("policy not found")
	}
	return s.policy, nil
}

func (s *stubCP) StartJob(_ context.Context, id uuid.UUID) error {
	s.started = append(s.started, id)
	return nil
}

func (s *stubCP) CompleteJob(_ context.Context, id uuid.UUID, _ string, _ int64, _ []string) error {
	s.completed = append(s.completed, id)
	return nil
}

func (s *stubCP) FailJob(_ context.Context, id uuid.UUID, reason string) error {
	s.failed = append(s.failed, failRecord{jobID: id, reason: reason})
	return nil
}

func (s *stubCP) ClaimNextRestoreTest(_ context.Context) (*catalog.RestoreTest, error) {
	return nil, catalog.ErrNotFound
}
func (s *stubCP) GetSnapshot(_ context.Context, _ uuid.UUID) (*catalog.Snapshot, error) {
	return nil, catalog.ErrNotFound
}
func (s *stubCP) CompleteRestoreTest(_ context.Context, _ uuid.UUID, _ int, _ int64) error {
	return nil
}
func (s *stubCP) FailRestoreTest(_ context.Context, _ uuid.UUID, _ string) error {
	return nil
}

// ── Tests ──────────────────────────────────────────────────────────────────

func TestAgent_FailsJob_WhenPolicyHasNoRepository(t *testing.T) {
	jobID := uuid.New()
	policyID := uuid.New()

	cp := &stubCP{
		jobs: []catalog.BackupJob{{ID: jobID, SystemID: uuid.New(), PolicyID: policyID, Status: "pending"}},
		policy: &catalog.BackupPolicy{
			ID:           policyID,
			Name:         "no-repo",
			Engine:       "restic",
			RepositoryID: nil,
		},
	}

	runBriefly(t, cp)

	if len(cp.failed) == 0 {
		t.Fatal("expected job to be failed")
	}
	if cp.failed[0].jobID != jobID {
		t.Errorf("wrong job failed: want %s, got %s", jobID, cp.failed[0].jobID)
	}
	if cp.failed[0].reason == "" {
		t.Error("expected non-empty failure reason")
	}
}

func TestAgent_StartsJob_BeforeRunningBackup(t *testing.T) {
	jobID := uuid.New()
	repoID := uuid.New()

	cp := &stubCP{
		jobs: []catalog.BackupJob{{ID: jobID, SystemID: uuid.New(), PolicyID: uuid.New(), Status: "pending"}},
		policy: &catalog.BackupPolicy{
			ID:           uuid.New(),
			Name:         "test",
			Engine:       "restic-nonexistent",
			RepositoryID: &repoID,
			Includes:     []string{"/tmp"},
		},
	}

	runBriefly(t, cp)

	if len(cp.started) == 0 {
		t.Fatal("expected StartJob to be called before backup")
	}
	if cp.started[0] != jobID {
		t.Errorf("StartJob: want %s, got %s", jobID, cp.started[0])
	}
}

func TestAgent_StopsOnUnauthorized(t *testing.T) {
	cp := &stubCP{
		listErr: agentclient.ErrUnauthorized,
	}

	cfg := agent.Config{
		PollInterval:   time.Hour,
		ResticBin:      "restic-nonexistent",
		ResticPassword: "test",
		ResticRepo:     "s3:test",
	}
	a := agent.New(cfg, cp, testLog)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	err := a.Run(ctx)
	if !errors.Is(err, agentclient.ErrUnauthorized) {
		t.Errorf("want ErrUnauthorized, got %v", err)
	}
}

func TestAgent_ContinuesPollOnTransientError(t *testing.T) {
	callCount := 0
	cp := &stubCP{}
	// First call fails, second succeeds with no jobs
	origJobs := cp.jobs
	_ = origJobs

	cfg := agent.Config{
		PollInterval:   20 * time.Millisecond,
		ResticBin:      "restic-nonexistent",
		ResticPassword: "test",
		ResticRepo:     "s3:test",
	}

	// Use a stub that fails once then returns empty
	failingCP := &failOnceCP{inner: cp, failCount: 1}
	_ = callCount
	a := agent.New(cfg, failingCP, testLog)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Should not return an error for transient failures
	err := a.Run(ctx)
	if err != nil && !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("unexpected error for transient failure: %v", err)
	}
}

type failOnceCP struct {
	inner     *stubCP
	failCount int
	calls     int
}

func (f *failOnceCP) ListPendingJobs(ctx context.Context) ([]catalog.BackupJob, error) {
	f.calls++
	if f.calls <= f.failCount {
		return nil, errors.New("transient network error")
	}
	return f.inner.ListPendingJobs(ctx)
}
func (f *failOnceCP) GetPolicy(ctx context.Context, id uuid.UUID) (*catalog.BackupPolicy, error) {
	return f.inner.GetPolicy(ctx, id)
}
func (f *failOnceCP) StartJob(ctx context.Context, id uuid.UUID) error {
	return f.inner.StartJob(ctx, id)
}
func (f *failOnceCP) CompleteJob(ctx context.Context, id uuid.UUID, s string, b int64, p []string) error {
	return f.inner.CompleteJob(ctx, id, s, b, p)
}
func (f *failOnceCP) FailJob(ctx context.Context, id uuid.UUID, r string) error {
	return f.inner.FailJob(ctx, id, r)
}
func (f *failOnceCP) ClaimNextRestoreTest(ctx context.Context) (*catalog.RestoreTest, error) {
	return f.inner.ClaimNextRestoreTest(ctx)
}
func (f *failOnceCP) GetSnapshot(ctx context.Context, id uuid.UUID) (*catalog.Snapshot, error) {
	return f.inner.GetSnapshot(ctx, id)
}
func (f *failOnceCP) CompleteRestoreTest(ctx context.Context, id uuid.UUID, fi int, b int64) error {
	return f.inner.CompleteRestoreTest(ctx, id, fi, b)
}
func (f *failOnceCP) FailRestoreTest(ctx context.Context, id uuid.UUID, r string) error {
	return f.inner.FailRestoreTest(ctx, id, r)
}

func runBriefly(t *testing.T, cp agent.ControlPlaneClient) {
	t.Helper()
	cfg := agent.Config{
		PollInterval:   time.Hour,
		ResticBin:      "restic-nonexistent",
		ResticPassword: "test",
		ResticRepo:     "s3:test",
	}
	a := agent.New(cfg, cp, testLog)
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	a.Run(ctx) //nolint:errcheck // context timeout is expected here
}
