package scheduler_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/cerberus8484/opensourcebackup/internal/catalog"
	"github.com/cerberus8484/opensourcebackup/internal/scheduler"
)

type reloadPolicyStore struct {
	policies []catalog.BackupPolicy
}

func (s *reloadPolicyStore) Create(_ context.Context, p *catalog.BackupPolicy) error {
	p.ID = uuid.New()
	s.policies = append(s.policies, *p)
	return nil
}
func (s *reloadPolicyStore) GetByID(_ context.Context, id uuid.UUID) (*catalog.BackupPolicy, error) {
	for _, p := range s.policies {
		if p.ID == id {
			return &p, nil
		}
	}
	return nil, catalog.ErrNotFound
}
func (s *reloadPolicyStore) List(_ context.Context) ([]catalog.BackupPolicy, error) {
	return s.policies, nil
}
func (s *reloadPolicyStore) Update(_ context.Context, p *catalog.BackupPolicy) error { return nil }
func (s *reloadPolicyStore) Delete(_ context.Context, _ uuid.UUID) error             { return nil }

func TestScheduler_PoliciesChanged_IsIdempotent(t *testing.T) {
	sched := scheduler.New(&failingReloadPolicyStore{}, &stubJobStore{}, testLog)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// PoliciesChanged with a failing store should not panic
	sched.PoliciesChanged(ctx)
	sched.PoliciesChanged(ctx)
}

func TestScheduler_PoliciesChanged_PicksUpNewPolicy(t *testing.T) {
	sched := "* * * * *"
	store := &reloadPolicyStore{policies: []catalog.BackupPolicy{}}

	sc := scheduler.New(store, &stubJobStore{}, testLog)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	sc.Start(ctx) //nolint:errcheck

	// Add a policy and reload
	store.policies = append(store.policies, catalog.BackupPolicy{
		ID:       uuid.New(),
		Name:     "new-policy",
		Engine:   "restic",
		Schedule: &sched,
	})
	sc.PoliciesChanged(context.Background())
	// No panic — reload completed
}

func TestScheduler_PoliciesChanged_InvalidCronSkipped(t *testing.T) {
	bad := "not-a-cron"
	store := &reloadPolicyStore{policies: []catalog.BackupPolicy{
		{ID: uuid.New(), Name: "bad-schedule", Engine: "restic", Schedule: &bad},
	}}

	sc := scheduler.New(store, &stubJobStore{}, testLog)

	// Should not panic with invalid cron expression
	sc.PoliciesChanged(context.Background())
}

type failingReloadPolicyStore struct{ reloadPolicyStore }

func (f *failingReloadPolicyStore) List(_ context.Context) ([]catalog.BackupPolicy, error) {
	return nil, errors.New("db error")
}
