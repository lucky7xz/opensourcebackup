package agent

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/cerberus8484/opensourcebackup/internal/agent/restic"
	"github.com/cerberus8484/opensourcebackup/internal/catalog"
)

// ControlPlaneClient is the interface the agent uses to talk to the control plane.
// Defined here so the agent package does not import the concrete HTTP client (DIP).
type ControlPlaneClient interface {
	ListPendingJobs(ctx context.Context, systemID uuid.UUID) ([]catalog.BackupJob, error)
	GetPolicy(ctx context.Context, id uuid.UUID) (*catalog.BackupPolicy, error)
	UpdateJobStatus(ctx context.Context, job *catalog.BackupJob) error
	CreateSnapshot(ctx context.Context, s *catalog.Snapshot) error
}

// Config holds all runtime parameters for the agent.
type Config struct {
	SystemID       uuid.UUID
	PollInterval   time.Duration
	ResticBin      string
	ResticPassword string
	ResticRepo     string
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

// Run starts the poll loop and blocks until ctx is canceled.
func (a *Agent) Run(ctx context.Context) error {
	a.log.Info("agent started",
		"system_id", a.cfg.SystemID,
		"poll_interval", a.cfg.PollInterval,
	)
	ticker := time.NewTicker(a.cfg.PollInterval)
	defer ticker.Stop()

	// Run once immediately, then on each tick.
	a.poll(ctx)
	for {
		select {
		case <-ctx.Done():
			a.log.Info("agent stopped")
			return nil
		case <-ticker.C:
			a.poll(ctx)
		}
	}
}

func (a *Agent) poll(ctx context.Context) {
	jobs, err := a.cp.ListPendingJobs(ctx, a.cfg.SystemID)
	if err != nil {
		a.log.Warn("poll failed", "error", err)
		return
	}
	for _, job := range jobs {
		a.executeJob(ctx, job)
	}
}

func (a *Agent) executeJob(ctx context.Context, job catalog.BackupJob) {
	log := a.log.With("job_id", job.ID, "policy_id", job.PolicyID)

	policy, err := a.cp.GetPolicy(ctx, job.PolicyID)
	if err != nil {
		log.Error("get policy failed", "error", err)
		a.failJob(ctx, &job, fmt.Sprintf("get policy: %s", err))
		return
	}

	job.Status = "running"
	now := time.Now()
	job.StartedAt = &now
	if err := a.cp.UpdateJobStatus(ctx, &job); err != nil {
		log.Error("mark running failed", "error", err)
		return
	}
	log.Info("job started", "engine", policy.Engine, "includes", policy.Includes)

	result, err := a.runner.Backup(ctx, restic.BackupOptions{
		Repo:     a.cfg.ResticRepo,
		Password: a.cfg.ResticPassword,
		Includes: policy.Includes,
		Excludes: policy.Excludes,
		Tags:     []string{fmt.Sprintf("system=%s", a.cfg.SystemID), fmt.Sprintf("policy=%s", policy.ID)},
	})
	if err != nil {
		log.Error("backup failed", "error", err)
		a.failJob(ctx, &job, err.Error())
		return
	}

	finished := time.Now()
	job.Status = "success"
	job.FinishedAt = &finished
	job.BytesUploaded = &result.BytesAdded
	if err := a.cp.UpdateJobStatus(ctx, &job); err != nil {
		log.Error("update job success failed", "error", err)
	}

	snap := &catalog.Snapshot{
		JobID:            job.ID,
		RepositoryID:     uuid.Nil, // set from policy/repo config in B12
		EngineSnapshotID: result.SnapshotID,
		Hostname:         strPtr(getHostname()),
		Paths:            policy.Includes,
		ChecksumStatus:   "unverified",
	}
	if err := a.cp.CreateSnapshot(ctx, snap); err != nil {
		log.Error("register snapshot failed", "error", err)
	}

	log.Info("job completed",
		"snapshot_id", result.SnapshotID,
		"bytes_added", result.BytesAdded,
		"files_new", result.FilesNew,
		"files_changed", result.FilesChanged,
	)
}

func (a *Agent) failJob(ctx context.Context, job *catalog.BackupJob, reason string) {
	now := time.Now()
	job.Status = "failed"
	job.FinishedAt = &now
	job.ErrorSummary = &reason
	if err := a.cp.UpdateJobStatus(ctx, job); err != nil {
		a.log.Error("fail job update failed", "error", err)
	}
}
