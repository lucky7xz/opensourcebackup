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
