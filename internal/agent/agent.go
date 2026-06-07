package agent

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"time"

	"github.com/google/uuid"

	agentclient "github.com/cerberus8484/opensourcebackup/internal/agent/client"
	"github.com/cerberus8484/opensourcebackup/internal/agent/restic"
	"github.com/cerberus8484/opensourcebackup/internal/catalog"

)

// ControlPlaneClient is the interface the agent uses to talk to the control plane.
// Defined here (DIP) so the agent package does not import the concrete HTTP client.
type ControlPlaneClient interface {
	// Heartbeat — called on every poll cycle to update last_seen_at on the control plane.
	// Returns ErrUnauthorized if the token was revoked (system deleted, re-enroll required).
	Heartbeat(ctx context.Context) error
	// Backup jobs
	ListPendingJobs(ctx context.Context) ([]catalog.BackupJob, error)
	GetPolicy(ctx context.Context, id uuid.UUID) (*catalog.BackupPolicy, error)
	StartJob(ctx context.Context, jobID uuid.UUID) error
	CompleteJob(ctx context.Context, jobID uuid.UUID, snapshotID string, bytesUploaded int64, paths []string) error
	FailJob(ctx context.Context, jobID uuid.UUID, reason string) error
	// ReportProgress sends a live progress snapshot while a backup runs (B_JOB_PROGRESS).
	ReportProgress(ctx context.Context, jobID uuid.UUID, p catalog.JobProgress) error
	// Repository lookup (agent reads location from policy)
	GetRepository(ctx context.Context, id uuid.UUID) (*catalog.BackupRepository, error)
	// Restore tests
	ClaimNextRestoreTest(ctx context.Context) (*catalog.RestoreTest, error)
	GetSnapshot(ctx context.Context, id uuid.UUID) (*catalog.Snapshot, error)
	CompleteRestoreTest(ctx context.Context, id uuid.UUID, files int, bytes int64) error
	FailRestoreTest(ctx context.Context, id uuid.UUID, reason string) error
}

// Config holds all runtime parameters for the agent.
type Config struct {
	PollInterval    time.Duration
	ResticBin       string
	ResticPassword  string
	ResticRepo      string
	RestoreTestRoot string // sandbox root for restore tests
}

// Agent polls the control plane for pending jobs and executes them.
type Agent struct {
	cfg    Config
	cp     ControlPlaneClient
	runner *restic.Runner
	log    *slog.Logger
}

// New creates an Agent.
func New(cfg Config, cp ControlPlaneClient, log *slog.Logger) *Agent {
	return &Agent{
		cfg:    cfg,
		cp:     cp,
		runner: restic.New(cfg.ResticBin),
		log:    log,
	}
}

// Run starts the poll loop and blocks until ctx is canceled or a fatal error occurs.
// Returns a non-nil error if the agent token is rejected (re-enrollment required).
func (a *Agent) Run(ctx context.Context) error {
	a.log.Info("agent started", "poll_interval", a.cfg.PollInterval)
	a.log.Info("restic process tuning applied", "mode", restic.ProcessTuningMode())

	ticker := time.NewTicker(a.cfg.PollInterval)
	defer ticker.Stop()

	// Poll once immediately, then on each tick.
	if err := a.poll(ctx); err != nil {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			a.log.Info("agent stopped")
			return nil
		case <-ticker.C:
			if err := a.poll(ctx); err != nil {
				return err
			}
		}
	}
}

func (a *Agent) poll(ctx context.Context) error {
	// Heartbeat first — updates last_seen on the control plane.
	// On 401 the token was revoked; stop immediately.
	if err := a.cp.Heartbeat(ctx); err != nil {
		if isUnauthorized(err) {
			a.log.Error("heartbeat rejected — token revoked, re-enrollment required")
			return err
		}
		// Network errors are non-fatal — agent continues working even if
		// the control plane is temporarily unreachable.
		a.log.Warn("heartbeat failed", "error", err)
	}

	// Poll backup jobs
	jobs, err := a.cp.ListPendingJobs(ctx)
	if err != nil {
		if isUnauthorized(err) {
			a.log.Error("agent token rejected — re-enrollment required")
			return err
		}
		a.log.Warn("poll jobs failed", "error", err)
		return nil
	}
	for _, job := range jobs {
		switch job.Type {
		case catalog.JobTypeRetention:
			// Retention jobs are handled by the retention pipeline
		case "verify":
			a.executeVerifyJob(ctx, job)
		default:
			a.executeJob(ctx, job)
		}
	}

	// Poll restore tests
	if err := a.executeNextRestoreTest(ctx); err != nil && isUnauthorized(err) {
		return err
	}
	return nil
}

// executeVerifyJob runs restic check for a verify job.
func (a *Agent) executeVerifyJob(ctx context.Context, job catalog.BackupJob) {
	log := a.log.With("job_id", job.ID, "policy_id", job.PolicyID)
	log.Info("verify job started")

	policy, err := a.cp.GetPolicy(ctx, job.PolicyID)
	if err != nil {
		log.Error("get policy failed", "error", err)
		a.doFail(ctx, log, job.ID, fmt.Sprintf("get policy: %s", err))
		return
	}

	repo, err := a.cp.GetRepository(ctx, *policy.RepositoryID)
	if err != nil {
		log.Error("get repository failed", "error", err)
		a.doFail(ctx, log, job.ID, fmt.Sprintf("get repository: %s", err))
		return
	}

	if err := a.cp.StartJob(ctx, job.ID); err != nil {
		log.Error("start job failed", "error", err)
		return
	}

	err = a.runner.Verify(ctx, restic.VerifyOptions{
		Repo:     repo.Location,
		Password: a.cfg.ResticPassword,
		ReadData: false, // fast check by default
	})
	if err != nil {
		log.Error("verify failed", "error", err)
		a.doFail(ctx, log, job.ID, err.Error())
		return
	}

	// Complete with zero bytes (verify doesn't transfer data)
	if err := a.cp.CompleteJob(ctx, job.ID, "", 0, nil); err != nil {
		log.Error("complete verify job failed", "error", err)
		return
	}
	log.Info("verify job completed")
}

func (a *Agent) executeNextRestoreTest(ctx context.Context) error {
	rt, err := a.cp.ClaimNextRestoreTest(ctx)
	if err != nil {
		if isUnauthorized(err) {
			return err
		}
		// ErrNotFound = no pending restore test — normal
		return nil
	}

	log := a.log.With("restore_test_id", rt.ID, "snapshot_id", rt.SnapshotID)
	log.Info("restore test started")

	root := a.cfg.RestoreTestRoot
	if root == "" {
		root = filepath.Join("data", "restore-tests")
	}
	target := filepath.Join(root, rt.ID.String())
	if rt.TargetPath != nil && *rt.TargetPath != "" {
		target = *rt.TargetPath
	}

	snap, err := a.cp.GetSnapshot(ctx, rt.SnapshotID)
	if err != nil {
		log.Error("could not load snapshot for restore test", "error", err)
		a.cp.FailRestoreTest(ctx, rt.ID, fmt.Sprintf("snapshot lookup: %s", err)) //nolint:errcheck
		return nil
	}

	// Use the repository that the snapshot actually belongs to, not the
	// agent's default configured repo — they may differ.
	repo, err := a.cp.GetRepository(ctx, rt.RepositoryID)
	if err != nil {
		log.Error("could not load repository for restore test", "error", err)
		a.cp.FailRestoreTest(ctx, rt.ID, fmt.Sprintf("repository lookup: %s", err)) //nolint:errcheck
		return nil
	}

	// If the user explicitly set a custom target path, skip the RestoreRoot
	// sandbox constraint — the user knows where they want to restore.
	restoreRoot := root
	if rt.TargetPath != nil && *rt.TargetPath != "" {
		restoreRoot = ""
	}

	result, err := a.runner.Restore(ctx, restic.RestoreOptions{
		Repo:        repo.Location,
		Password:    a.cfg.ResticPassword,
		SnapshotID:  snap.EngineSnapshotID,
		TargetPath:  target,
		RestoreRoot: restoreRoot,
	})
	if err != nil {
		log.Error("restore test failed", "error", err)
		if failErr := a.cp.FailRestoreTest(ctx, rt.ID, err.Error()); failErr != nil {
			log.Error("fail restore test update failed", "error", failErr)
		}
		return nil
	}

	if err := a.cp.CompleteRestoreTest(ctx, rt.ID, result.VerifiedFiles, result.VerifiedBytes); err != nil {
		log.Error("complete restore test update failed", "error", err)
		return nil
	}
	log.Info("restore test completed",
		"verified_files", result.VerifiedFiles,
		"verified_bytes", result.VerifiedBytes,
		"target", result.TargetPath,
	)
	return nil
}

func (a *Agent) executeJob(ctx context.Context, job catalog.BackupJob) {
	log := a.log.With("job_id", job.ID, "policy_id", job.PolicyID)

	policy, err := a.cp.GetPolicy(ctx, job.PolicyID)
	if err != nil {
		log.Error("get policy failed", "error", err)
		a.doFail(ctx, log, job.ID, fmt.Sprintf("get policy: %s", err))
		return
	}

	if policy.RepositoryID == nil {
		a.doFail(ctx, log, job.ID, "policy has no repository configured")
		log.Error("policy has no repository_id — cannot determine backup target")
		return
	}

	// Fetch the repository to get the actual backup destination (Location).
	// This allows each policy to target a different repo — NAS, S3, local path, etc.
	repo, err := a.cp.GetRepository(ctx, *policy.RepositoryID)
	if err != nil {
		log.Error("could not load repository", "error", err)
		a.doFail(ctx, log, job.ID, fmt.Sprintf("repository lookup: %s", err))
		return
	}
	// Use policy repository location; fall back to agent env var if empty.
	repoLocation := repo.Location
	if repoLocation == "" {
		repoLocation = a.cfg.ResticRepo
	}

	if err := a.cp.StartJob(ctx, job.ID); err != nil {
		log.Error("mark job running failed", "error", err)
		return
	}
	log.Info("job started",
		"engine", policy.Engine,
		"includes", policy.Includes,
		"repository", repoLocation,
	)

	// Live progress reporting (B_JOB_PROGRESS): throttle to ~2s and compute
	// throughput from the byte delta. Best-effort — never fails the backup.
	var (
		lastReport time.Time
		lastBytes  int64
	)
	onProgress := func(p restic.Progress) {
		now := time.Now()
		if !lastReport.IsZero() && now.Sub(lastReport) < 2*time.Second {
			return
		}
		var bps int64
		if !lastReport.IsZero() {
			if dt := now.Sub(lastReport).Seconds(); dt > 0 && p.BytesDone >= lastBytes {
				bps = int64(float64(p.BytesDone-lastBytes) / dt)
			}
		}
		lastReport = now
		lastBytes = p.BytesDone
		if err := a.cp.ReportProgress(ctx, job.ID, catalog.JobProgress{
			Phase:         p.Phase,
			Percent:       p.Percent,
			BytesDone:     p.BytesDone,
			BytesTotal:    p.TotalBytes,
			FilesDone:     p.FilesDone,
			FilesTotal:    p.TotalFiles,
			ThroughputBps: bps,
		}); err != nil {
			log.Debug("report progress failed (best-effort)", "error", err)
		}
	}

	result, err := a.runner.Backup(ctx, restic.BackupOptions{
		Repo:       repoLocation,
		Password:   a.cfg.ResticPassword,
		Includes:   policy.Includes,
		Excludes:   policy.Excludes,
		Tags:       []string{fmt.Sprintf("policy=%s", policy.ID)},
		OnProgress: onProgress,
	})
	if err != nil {
		log.Error("backup failed", "error", err)
		a.doFail(ctx, log, job.ID, err.Error())
		return
	}

	if err := a.cp.CompleteJob(ctx, job.ID, result.SnapshotID, result.BytesAdded, policy.Includes); err != nil {
		log.Error("complete job failed", "error", err)
		return
	}

	log.Info("job completed",
		"snapshot_id", result.SnapshotID,
		"bytes_added", result.BytesAdded,
		"files_new", result.FilesNew,
		"files_changed", result.FilesChanged,
	)
}

func (a *Agent) doFail(ctx context.Context, log *slog.Logger, jobID uuid.UUID, reason string) {
	if err := a.cp.FailJob(ctx, jobID, reason); err != nil {
		log.Error("fail job update failed", "error", err)
	}
}

func isUnauthorized(err error) bool {
	return errors.Is(err, agentclient.ErrUnauthorized)
}
