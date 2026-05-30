package api_test

import (
	"context"

	"github.com/cerberus8484/opensourcebackup/internal/catalog"
	"github.com/google/uuid"
)

// stubSystemStore is an in-memory SystemStore for unit tests.
type stubSystemStore struct {
	systems map[uuid.UUID]*catalog.System
}

func newStubSystemStore() *stubSystemStore {
	return &stubSystemStore{systems: make(map[uuid.UUID]*catalog.System)}
}

func (s *stubSystemStore) Create(_ context.Context, sys *catalog.System) error {
	sys.ID = uuid.New()
	cp := *sys
	s.systems[sys.ID] = &cp
	return nil
}

func (s *stubSystemStore) GetByID(_ context.Context, id uuid.UUID) (*catalog.System, error) {
	sys, ok := s.systems[id]
	if !ok {
		return nil, catalog.ErrNotFound
	}
	cp := *sys
	return &cp, nil
}

func (s *stubSystemStore) List(_ context.Context) ([]catalog.System, error) {
	out := make([]catalog.System, 0, len(s.systems))
	for _, sys := range s.systems {
		out = append(out, *sys)
	}
	return out, nil
}

func (s *stubSystemStore) Update(_ context.Context, sys *catalog.System) error {
	if _, ok := s.systems[sys.ID]; !ok {
		return catalog.ErrNotFound
	}
	cp := *sys
	s.systems[sys.ID] = &cp
	return nil
}

func (s *stubSystemStore) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := s.systems[id]; !ok {
		return catalog.ErrNotFound
	}
	delete(s.systems, id)
	return nil
}

// stubRepositoryStore is an in-memory RepositoryStore for unit tests.
type stubRepositoryStore struct {
	repos map[uuid.UUID]*catalog.BackupRepository
}

func newStubRepositoryStore() *stubRepositoryStore {
	return &stubRepositoryStore{repos: make(map[uuid.UUID]*catalog.BackupRepository)}
}

func (s *stubRepositoryStore) Create(_ context.Context, r *catalog.BackupRepository) error {
	r.ID = uuid.New()
	cp := *r
	s.repos[r.ID] = &cp
	return nil
}

func (s *stubRepositoryStore) GetByID(_ context.Context, id uuid.UUID) (*catalog.BackupRepository, error) {
	r, ok := s.repos[id]
	if !ok {
		return nil, catalog.ErrNotFound
	}
	cp := *r
	return &cp, nil
}

func (s *stubRepositoryStore) List(_ context.Context) ([]catalog.BackupRepository, error) {
	out := make([]catalog.BackupRepository, 0, len(s.repos))
	for _, r := range s.repos {
		out = append(out, *r)
	}
	return out, nil
}

func (s *stubRepositoryStore) Update(_ context.Context, r *catalog.BackupRepository) error {
	if _, ok := s.repos[r.ID]; !ok {
		return catalog.ErrNotFound
	}
	cp := *r
	s.repos[r.ID] = &cp
	return nil
}

func (s *stubRepositoryStore) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := s.repos[id]; !ok {
		return catalog.ErrNotFound
	}
	delete(s.repos, id)
	return nil
}

// stubJobStore is an in-memory JobStore for unit tests.
type stubJobStore struct {
	jobs map[uuid.UUID]*catalog.BackupJob
}

func newStubJobStore() *stubJobStore {
	return &stubJobStore{jobs: make(map[uuid.UUID]*catalog.BackupJob)}
}

func (s *stubJobStore) Create(_ context.Context, j *catalog.BackupJob) error {
	j.ID = uuid.New()
	cp := *j
	s.jobs[j.ID] = &cp
	return nil
}

func (s *stubJobStore) GetByID(_ context.Context, id uuid.UUID) (*catalog.BackupJob, error) {
	j, ok := s.jobs[id]
	if !ok {
		return nil, catalog.ErrNotFound
	}
	cp := *j
	return &cp, nil
}

func (s *stubJobStore) List(_ context.Context) ([]catalog.BackupJob, error) {
	out := make([]catalog.BackupJob, 0, len(s.jobs))
	for _, j := range s.jobs {
		out = append(out, *j)
	}
	return out, nil
}

func (s *stubJobStore) LatestByPolicyID(_ context.Context, _ uuid.UUID) (*catalog.BackupJob, error) {
	return nil, catalog.ErrNotFound
}

func (s *stubJobStore) ListBySystemID(_ context.Context, systemID uuid.UUID) ([]catalog.BackupJob, error) {
	var out []catalog.BackupJob
	for _, j := range s.jobs {
		if j.SystemID == systemID {
			out = append(out, *j)
		}
	}
	return out, nil
}

func (s *stubJobStore) Update(_ context.Context, j *catalog.BackupJob) error {
	if _, ok := s.jobs[j.ID]; !ok {
		return catalog.ErrNotFound
	}
	cp := *j
	s.jobs[j.ID] = &cp
	return nil
}

func (s *stubJobStore) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := s.jobs[id]; !ok {
		return catalog.ErrNotFound
	}
	delete(s.jobs, id)
	return nil
}

// stubSnapshotStore is an in-memory SnapshotStore for unit tests.
type stubSnapshotStore struct {
	snaps map[uuid.UUID]*catalog.Snapshot
}

func newStubSnapshotStore() *stubSnapshotStore {
	return &stubSnapshotStore{snaps: make(map[uuid.UUID]*catalog.Snapshot)}
}

func (s *stubSnapshotStore) Create(_ context.Context, snap *catalog.Snapshot) error {
	snap.ID = uuid.New()
	cp := *snap
	s.snaps[snap.ID] = &cp
	return nil
}

func (s *stubSnapshotStore) GetByID(_ context.Context, id uuid.UUID) (*catalog.Snapshot, error) {
	snap, ok := s.snaps[id]
	if !ok {
		return nil, catalog.ErrNotFound
	}
	cp := *snap
	return &cp, nil
}

func (s *stubSnapshotStore) List(_ context.Context) ([]catalog.Snapshot, error) {
	out := make([]catalog.Snapshot, 0, len(s.snaps))
	for _, snap := range s.snaps {
		out = append(out, *snap)
	}
	return out, nil
}

func (s *stubSnapshotStore) ListByJobID(_ context.Context, jobID uuid.UUID) ([]catalog.Snapshot, error) {
	var out []catalog.Snapshot
	for _, snap := range s.snaps {
		if snap.JobID == jobID {
			out = append(out, *snap)
		}
	}
	return out, nil
}

func (s *stubSnapshotStore) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := s.snaps[id]; !ok {
		return catalog.ErrNotFound
	}
	delete(s.snaps, id)
	return nil
}

// stubPolicyStore is an in-memory PolicyStore for unit tests.
type stubPolicyStore struct {
	policies map[uuid.UUID]*catalog.BackupPolicy
}

func newStubPolicyStore() *stubPolicyStore {
	return &stubPolicyStore{policies: make(map[uuid.UUID]*catalog.BackupPolicy)}
}

func (s *stubPolicyStore) Create(_ context.Context, p *catalog.BackupPolicy) error {
	p.ID = uuid.New()
	cp := *p
	s.policies[p.ID] = &cp
	return nil
}

func (s *stubPolicyStore) GetByID(_ context.Context, id uuid.UUID) (*catalog.BackupPolicy, error) {
	p, ok := s.policies[id]
	if !ok {
		return nil, catalog.ErrNotFound
	}
	cp := *p
	return &cp, nil
}

func (s *stubPolicyStore) List(_ context.Context) ([]catalog.BackupPolicy, error) {
	out := make([]catalog.BackupPolicy, 0, len(s.policies))
	for _, p := range s.policies {
		out = append(out, *p)
	}
	return out, nil
}

func (s *stubPolicyStore) Update(_ context.Context, p *catalog.BackupPolicy) error {
	if _, ok := s.policies[p.ID]; !ok {
		return catalog.ErrNotFound
	}
	cp := *p
	s.policies[p.ID] = &cp
	return nil
}

func (s *stubPolicyStore) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := s.policies[id]; !ok {
		return catalog.ErrNotFound
	}
	delete(s.policies, id)
	return nil
}
