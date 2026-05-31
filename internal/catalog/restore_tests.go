package catalog

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// RestoreTestStore manages restore_tests records.
type RestoreTestStore interface {
	Create(ctx context.Context, rt *RestoreTest) error
	GetByID(ctx context.Context, id uuid.UUID) (*RestoreTest, error)
	List(ctx context.Context) ([]RestoreTest, error)
	ListBySnapshotID(ctx context.Context, snapshotID uuid.UUID) ([]RestoreTest, error)
	ListBySystemID(ctx context.Context, systemID uuid.UUID) ([]RestoreTest, error)
	Update(ctx context.Context, rt *RestoreTest) error
	Delete(ctx context.Context, id uuid.UUID) error
	HasSuccessfulTest(ctx context.Context, snapshotID uuid.UUID) (bool, error)
}

type pgRestoreTestStore struct {
	db *DB
}

// NewRestoreTestStore returns a PostgreSQL-backed RestoreTestStore.
func NewRestoreTestStore(db *DB) RestoreTestStore {
	return &pgRestoreTestStore{db: db}
}

func (s *pgRestoreTestStore) Create(ctx context.Context, rt *RestoreTest) error {
	row := s.db.pool.QueryRow(ctx, `
		INSERT INTO restore_tests
		  (snapshot_id, system_id, repository_id, status, target_path)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at`,
		pgtype.UUID{Bytes: rt.SnapshotID, Valid: true},
		pgtype.UUID{Bytes: rt.SystemID, Valid: true},
		pgtype.UUID{Bytes: rt.RepositoryID, Valid: true},
		rt.Status, rt.TargetPath,
	)
	var rawID pgtype.UUID
	if err := row.Scan(&rawID, &rt.CreatedAt, &rt.UpdatedAt); err != nil {
		return err
	}
	rt.ID = uuid.UUID(rawID.Bytes)
	return nil
}

func (s *pgRestoreTestStore) GetByID(ctx context.Context, id uuid.UUID) (*RestoreTest, error) {
	row := s.db.pool.QueryRow(ctx, `
		SELECT id, snapshot_id, system_id, repository_id, status, target_path,
		       started_at, finished_at, verified_files, verified_bytes, error_summary,
		       created_at, updated_at
		FROM restore_tests WHERE id = $1`,
		pgtype.UUID{Bytes: id, Valid: true},
	)
	rt, err := scanRestoreTest(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return rt, err
}

func (s *pgRestoreTestStore) List(ctx context.Context) ([]RestoreTest, error) {
	return s.query(ctx, `
		SELECT id, snapshot_id, system_id, repository_id, status, target_path,
		       started_at, finished_at, verified_files, verified_bytes, error_summary,
		       created_at, updated_at
		FROM restore_tests ORDER BY created_at DESC`)
}

func (s *pgRestoreTestStore) ListBySnapshotID(ctx context.Context, snapshotID uuid.UUID) ([]RestoreTest, error) {
	return s.query(ctx, `
		SELECT id, snapshot_id, system_id, repository_id, status, target_path,
		       started_at, finished_at, verified_files, verified_bytes, error_summary,
		       created_at, updated_at
		FROM restore_tests WHERE snapshot_id = $1 ORDER BY created_at DESC`,
		pgtype.UUID{Bytes: snapshotID, Valid: true},
	)
}

func (s *pgRestoreTestStore) ListBySystemID(ctx context.Context, systemID uuid.UUID) ([]RestoreTest, error) {
	return s.query(ctx, `
		SELECT id, snapshot_id, system_id, repository_id, status, target_path,
		       started_at, finished_at, verified_files, verified_bytes, error_summary,
		       created_at, updated_at
		FROM restore_tests WHERE system_id = $1 ORDER BY created_at DESC`,
		pgtype.UUID{Bytes: systemID, Valid: true},
	)
}

func (s *pgRestoreTestStore) Update(ctx context.Context, rt *RestoreTest) error {
	rt.UpdatedAt = time.Now()
	tag, err := s.db.pool.Exec(ctx, `
		UPDATE restore_tests
		SET status=$1, started_at=$2, finished_at=$3,
		    verified_files=$4, verified_bytes=$5, error_summary=$6, updated_at=$7
		WHERE id=$8`,
		rt.Status, rt.StartedAt, rt.FinishedAt,
		rt.VerifiedFiles, rt.VerifiedBytes, rt.ErrorSummary, rt.UpdatedAt,
		pgtype.UUID{Bytes: rt.ID, Valid: true},
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *pgRestoreTestStore) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := s.db.pool.Exec(ctx, `DELETE FROM restore_tests WHERE id = $1`,
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

// HasSuccessfulTest returns true if the snapshot has at least one successful restore test.
func (s *pgRestoreTestStore) HasSuccessfulTest(ctx context.Context, snapshotID uuid.UUID) (bool, error) {
	var count int
	err := s.db.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM restore_tests WHERE snapshot_id=$1 AND status='success'`,
		pgtype.UUID{Bytes: snapshotID, Valid: true},
	).Scan(&count)
	return count > 0, err
}

func (s *pgRestoreTestStore) query(ctx context.Context, sql string, args ...any) ([]RestoreTest, error) {
	rows, err := s.db.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tests []RestoreTest
	for rows.Next() {
		rt, err := scanRestoreTest(rows)
		if err != nil {
			return nil, err
		}
		tests = append(tests, *rt)
	}
	return tests, rows.Err()
}

func scanRestoreTest(row rowScanner) (*RestoreTest, error) {
	var (
		rt      RestoreTest
		rawID   pgtype.UUID
		rawSnap pgtype.UUID
		rawSys  pgtype.UUID
		rawRepo pgtype.UUID
	)
	if err := row.Scan(
		&rawID, &rawSnap, &rawSys, &rawRepo,
		&rt.Status, &rt.TargetPath,
		&rt.StartedAt, &rt.FinishedAt,
		&rt.VerifiedFiles, &rt.VerifiedBytes, &rt.ErrorSummary,
		&rt.CreatedAt, &rt.UpdatedAt,
	); err != nil {
		return nil, err
	}
	rt.ID = uuid.UUID(rawID.Bytes)
	rt.SnapshotID = uuid.UUID(rawSnap.Bytes)
	rt.SystemID = uuid.UUID(rawSys.Bytes)
	rt.RepositoryID = uuid.UUID(rawRepo.Bytes)
	return &rt, nil
}
