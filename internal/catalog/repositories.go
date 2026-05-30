package catalog

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// RepositoryStore defines data access for the repositories table.
type RepositoryStore interface {
	Create(ctx context.Context, r *BackupRepository) error
	GetByID(ctx context.Context, id uuid.UUID) (*BackupRepository, error)
	List(ctx context.Context) ([]BackupRepository, error)
	Update(ctx context.Context, r *BackupRepository) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type pgRepositoryStore struct {
	db *DB
}

// NewRepositoryStore returns a PostgreSQL-backed RepositoryStore.
func NewRepositoryStore(db *DB) RepositoryStore {
	return &pgRepositoryStore{db: db}
}

func (s *pgRepositoryStore) Create(ctx context.Context, r *BackupRepository) error {
	row := s.db.pool.QueryRow(ctx, `
		INSERT INTO repositories (type, location, encryption_mode, object_lock_enabled, retention_policy_id)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at`,
		r.Type, r.Location, r.EncryptionMode, r.ObjectLockEnabled, uuidPtrToRaw(r.RetentionPolicyID),
	)
	var rawID pgtype.UUID
	if err := row.Scan(&rawID, &r.CreatedAt); err != nil {
		return err
	}
	r.ID = uuid.UUID(rawID.Bytes)
	return nil
}

func (s *pgRepositoryStore) GetByID(ctx context.Context, id uuid.UUID) (*BackupRepository, error) {
	row := s.db.pool.QueryRow(ctx, `
		SELECT id, type, location, encryption_mode, object_lock_enabled, retention_policy_id, created_at
		FROM repositories WHERE id = $1`,
		pgtype.UUID{Bytes: id, Valid: true},
	)
	r, err := scanRepository(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return r, err
}

func (s *pgRepositoryStore) List(ctx context.Context) ([]BackupRepository, error) {
	rows, err := s.db.pool.Query(ctx, `
		SELECT id, type, location, encryption_mode, object_lock_enabled, retention_policy_id, created_at
		FROM repositories ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var repos []BackupRepository
	for rows.Next() {
		r, err := scanRepository(rows)
		if err != nil {
			return nil, err
		}
		repos = append(repos, *r)
	}
	return repos, rows.Err()
}

func (s *pgRepositoryStore) Update(ctx context.Context, r *BackupRepository) error {
	tag, err := s.db.pool.Exec(ctx, `
		UPDATE repositories
		SET type=$1, location=$2, encryption_mode=$3, object_lock_enabled=$4, retention_policy_id=$5
		WHERE id=$6`,
		r.Type, r.Location, r.EncryptionMode, r.ObjectLockEnabled,
		uuidPtrToRaw(r.RetentionPolicyID),
		pgtype.UUID{Bytes: r.ID, Valid: true},
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *pgRepositoryStore) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := s.db.pool.Exec(ctx, `DELETE FROM repositories WHERE id = $1`,
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

func scanRepository(row rowScanner) (*BackupRepository, error) {
	var (
		r       BackupRepository
		rawID   pgtype.UUID
		rawRetID pgtype.UUID
	)
	if err := row.Scan(
		&rawID, &r.Type, &r.Location, &r.EncryptionMode,
		&r.ObjectLockEnabled, &rawRetID, &r.CreatedAt,
	); err != nil {
		return nil, err
	}
	r.ID = uuid.UUID(rawID.Bytes)
	if rawRetID.Valid {
		id := uuid.UUID(rawRetID.Bytes)
		r.RetentionPolicyID = &id
	}
	return &r, nil
}

func uuidPtrToRaw(id *uuid.UUID) pgtype.UUID {
	if id == nil {
		return pgtype.UUID{Valid: false}
	}
	return pgtype.UUID{Bytes: *id, Valid: true}
}
