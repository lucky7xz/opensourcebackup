package catalog

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// PolicyStore defines data access for the backup_policies table.
type PolicyStore interface {
	Create(ctx context.Context, p *BackupPolicy) error
	GetByID(ctx context.Context, id uuid.UUID) (*BackupPolicy, error)
	List(ctx context.Context) ([]BackupPolicy, error)
	Update(ctx context.Context, p *BackupPolicy) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type pgPolicyStore struct {
	db *DB
}

// NewPolicyStore returns a PostgreSQL-backed PolicyStore.
func NewPolicyStore(db *DB) PolicyStore {
	return &pgPolicyStore{db: db}
}

func (s *pgPolicyStore) Create(ctx context.Context, p *BackupPolicy) error {
	retentionJSON, err := marshalJSONB(p.Retention)
	if err != nil {
		return err
	}
	row := s.db.pool.QueryRow(ctx, `
		INSERT INTO backup_policies
		  (name, includes, excludes, schedule, retention, engine, pre_hooks, post_hooks, repository_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at`,
		p.Name, orEmptySlice(p.Includes), orEmptySlice(p.Excludes), p.Schedule,
		retentionJSON, p.Engine, orEmptySlice(p.PreHooks), orEmptySlice(p.PostHooks),
		uuidPtrToRaw(p.RepositoryID),
	)
	var rawID pgtype.UUID
	if err := row.Scan(&rawID, &p.CreatedAt); err != nil {
		return err
	}
	p.ID = uuid.UUID(rawID.Bytes)
	return nil
}

func (s *pgPolicyStore) GetByID(ctx context.Context, id uuid.UUID) (*BackupPolicy, error) {
	row := s.db.pool.QueryRow(ctx, `
		SELECT id, name, includes, excludes, schedule, retention, engine, pre_hooks, post_hooks, repository_id, created_at
		FROM backup_policies WHERE id = $1`,
		pgtype.UUID{Bytes: id, Valid: true},
	)
	p, err := scanPolicy(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return p, err
}

func (s *pgPolicyStore) List(ctx context.Context) ([]BackupPolicy, error) {
	rows, err := s.db.pool.Query(ctx, `
		SELECT id, name, includes, excludes, schedule, retention, engine, pre_hooks, post_hooks, repository_id, created_at
		FROM backup_policies ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var policies []BackupPolicy
	for rows.Next() {
		p, err := scanPolicy(rows)
		if err != nil {
			return nil, err
		}
		policies = append(policies, *p)
	}
	return policies, rows.Err()
}

func (s *pgPolicyStore) Update(ctx context.Context, p *BackupPolicy) error {
	retentionJSON, err := marshalJSONB(p.Retention)
	if err != nil {
		return err
	}
	tag, err := s.db.pool.Exec(ctx, `
		UPDATE backup_policies
		SET name=$1, includes=$2, excludes=$3, schedule=$4,
		    retention=$5, engine=$6, pre_hooks=$7, post_hooks=$8, repository_id=$9
		WHERE id=$10`,
		p.Name, orEmptySlice(p.Includes), orEmptySlice(p.Excludes), p.Schedule,
		retentionJSON, p.Engine, orEmptySlice(p.PreHooks), orEmptySlice(p.PostHooks),
		uuidPtrToRaw(p.RepositoryID),
		pgtype.UUID{Bytes: p.ID, Valid: true},
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *pgPolicyStore) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := s.db.pool.Exec(ctx, `DELETE FROM backup_policies WHERE id = $1`,
		pgtype.UUID{Bytes: id, Valid: true},
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func orEmptySlice(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

func scanPolicy(row rowScanner) (*BackupPolicy, error) {
	var (
		p            BackupPolicy
		rawID        pgtype.UUID
		rawRepoID    pgtype.UUID
		retentionRaw []byte
	)
	if err := row.Scan(
		&rawID, &p.Name, &p.Includes, &p.Excludes, &p.Schedule,
		&retentionRaw, &p.Engine, &p.PreHooks, &p.PostHooks, &rawRepoID, &p.CreatedAt,
	); err != nil {
		return nil, err
	}
	p.ID = uuid.UUID(rawID.Bytes)
	if rawRepoID.Valid {
		id := uuid.UUID(rawRepoID.Bytes)
		p.RepositoryID = &id
	}
	if len(retentionRaw) > 0 {
		if err := json.Unmarshal(retentionRaw, &p.Retention); err != nil {
			return nil, err
		}
	}
	return &p, nil
}
