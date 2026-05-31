package scheduler

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/robfig/cron/v3"

	"github.com/cerberus8484/opensourcebackup/internal/catalog"
)

// PolicyChangeNotifier is implemented by the Scheduler.
// The API handler calls PoliciesChanged after any policy mutation so that
// the scheduler picks up the new state without a restart.
type PolicyChangeNotifier interface {
	PoliciesChanged(ctx context.Context)
}

// Scheduler dispatches backup jobs according to policy cron schedules.
type Scheduler struct {
	policies catalog.PolicyStore
	jobs     catalog.JobStore
	log      *slog.Logger

	mu   sync.Mutex
	cron *cron.Cron
}

// New creates a Scheduler. Call Start to activate it.
func New(policies catalog.PolicyStore, jobs catalog.JobStore, log *slog.Logger) *Scheduler {
	return &Scheduler{
		policies: policies,
		jobs:     jobs,
		log:      log,
		cron:     cron.New(),
	}
}

// Start loads all scheduled policies, registers cron entries, and runs the
// dead-man's switch checker until ctx is canceled.
func (s *Scheduler) Start(ctx context.Context) error {
	if err := s.reload(ctx); err != nil {
		return err
	}

	s.mu.Lock()
	s.cron.Start()
	s.mu.Unlock()

	go s.runDeadManSwitch(ctx)

	<-ctx.Done()
	s.mu.Lock()
	s.cron.Stop()
	s.mu.Unlock()
	s.log.Info("scheduler stopped")
	return nil
}

// PoliciesChanged implements PolicyChangeNotifier.
// Safe to call from any goroutine — reloads the cron schedule atomically.
func (s *Scheduler) PoliciesChanged(ctx context.Context) {
	s.log.Info("scheduler: policies changed — reloading")
	if err := s.reload(ctx); err != nil {
		s.log.Error("scheduler reload failed", "error", err)
	}
}

// reload stops the current cron, loads all policies from DB,
// and registers new cron entries. Thread-safe via mutex.
func (s *Scheduler) reload(ctx context.Context) error {
	policies, err := s.policies.List(ctx)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Stop old cron and create a fresh one.
	s.cron.Stop()
	s.cron = cron.New()

	scheduled := 0
	for _, p := range policies {
		if p.Schedule == nil || *p.Schedule == "" {
			continue
		}
		pol := p // capture for closure
		if _, err := s.cron.AddFunc(*pol.Schedule, func() {
			s.dispatchJob(context.Background(), pol)
		}); err != nil {
			s.log.Warn("invalid cron schedule — skipping policy",
				"policy_id", pol.ID,
				"policy_name", pol.Name,
				"schedule", *pol.Schedule,
				"error", err,
			)
			continue
		}
		scheduled++
	}

	s.cron.Start()
	s.log.Info("scheduler reloaded", "scheduled_policies", scheduled)
	return nil
}

// dispatchJob creates a pending BackupJob record for the agent to pick up.
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

func (s *Scheduler) runDeadManSwitch(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.mu.Lock()
			// Collect current entries to check
			s.mu.Unlock()
			s.checkDeadMan(ctx)
		}
	}
}

func (s *Scheduler) checkDeadMan(ctx context.Context) {
	policies, err := s.policies.List(ctx)
	if err != nil {
		s.log.Error("dead-man: failed to load policies", "error", err)
		return
	}
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
				"last_job_at", latest.CreatedAt,
			)
		}
	}
}

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
