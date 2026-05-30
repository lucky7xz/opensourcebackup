package scheduler

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/cerberus8484/opensourcebackup/internal/catalog"
	"github.com/robfig/cron/v3"
)

// Scheduler dispatches backup jobs according to policy cron schedules.
type Scheduler struct {
	policies catalog.PolicyStore
	jobs     catalog.JobStore
	cron     *cron.Cron
	log      *slog.Logger
}

// New creates a Scheduler. Call Start to activate it.
func New(policies catalog.PolicyStore, jobs catalog.JobStore, log *slog.Logger) *Scheduler {
	return &Scheduler{
		policies: policies,
		jobs:     jobs,
		cron:     cron.New(),
		log:      log,
	}
}

// Start loads all scheduled policies, registers cron entries, and runs the
// dead-man's switch checker until ctx is cancelled.
func (s *Scheduler) Start(ctx context.Context) error {
	policies, err := s.policies.List(ctx)
	if err != nil {
		return err
	}

	scheduled := 0
	for _, p := range policies {
		if p.Schedule == nil || *p.Schedule == "" {
			continue
		}
		pol := p // capture for closure
		_, err := s.cron.AddFunc(*pol.Schedule, func() {
			s.dispatchJob(context.Background(), pol)
		})
		if err != nil {
			s.log.Warn("invalid cron schedule — skipping policy",
				"policy_id", pol.ID,
				"policy_name", pol.Name,
				"schedule", *pol.Schedule,
				"error", err,
			)
			continue
		}
		scheduled++
		s.log.Info("policy scheduled",
			"policy_id", pol.ID,
			"policy_name", pol.Name,
			"schedule", *pol.Schedule,
		)
	}

	s.cron.Start()
	s.log.Info("scheduler started", "scheduled_policies", scheduled)

	go s.runDeadManSwitch(ctx, policies)

	<-ctx.Done()
	s.cron.Stop()
	s.log.Info("scheduler stopped")
	return nil
}

// dispatchJob creates a pending BackupJob record for the agent to pick up.
// SystemID is left zero — the agent fills this in when it claims the job.
func (s *Scheduler) dispatchJob(ctx context.Context, p catalog.BackupPolicy) {
	j := &catalog.BackupJob{
		PolicyID: p.ID,
		Status:   "pending",
	}
	if err := s.jobs.Create(ctx, j); err != nil {
		s.log.Error("failed to dispatch job",
			"policy_id", p.ID,
			"policy_name", p.Name,
			"error", err,
		)
		return
	}
	s.log.Info("job dispatched",
		"job_id", j.ID,
		"policy_id", p.ID,
		"policy_name", p.Name,
	)
}

func (s *Scheduler) runDeadManSwitch(ctx context.Context, policies []catalog.BackupPolicy) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.checkDeadMan(ctx, policies)
		}
	}
}

func (s *Scheduler) checkDeadMan(ctx context.Context, policies []catalog.BackupPolicy) {
	for _, p := range policies {
		if p.Schedule == nil || *p.Schedule == "" {
			continue
		}
		interval, err := cronInterval(*p.Schedule)
		if err != nil {
			continue
		}
		deadline := time.Now().Add(-time.Duration(float64(interval) * 1.5))

		latest, err := s.jobs.LatestByPolicyID(ctx, p.ID)
		if errors.Is(err, catalog.ErrNotFound) {
			s.log.Warn("dead-man: no job ever ran for policy",
				"policy_id", p.ID,
				"policy_name", p.Name,
				"expected_interval", interval,
			)
			continue
		}
		if err != nil {
			s.log.Error("dead-man: query failed", "policy_id", p.ID, "error", err)
			continue
		}
		if latest.CreatedAt.Before(deadline) {
			s.log.Warn("dead-man: overdue job detected",
				"policy_id", p.ID,
				"policy_name", p.Name,
				"last_job_id", latest.ID,
				"last_job_at", latest.CreatedAt,
				"overdue_since", deadline,
			)
		}
	}
}

// cronInterval returns the approximate duration between two consecutive
// scheduled times for the given cron expression.
func cronInterval(expr string) (time.Duration, error) {
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	sched, err := parser.Parse(expr)
	if err != nil {
		return 0, err
	}
	now := time.Now()
	next1 := sched.Next(now)
	next2 := sched.Next(next1)
	return next2.Sub(next1), nil
}
