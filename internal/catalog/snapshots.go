package catalog

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// SnapshotStore defines data access for the snapshots table.
type SnapshotStore interface {
	Create(ctx context.Context, s *Snapshot) error
	GetByID(ctx context.Context, id uuid.UUID) (*Snapshot, error)
	List(ctx context.Context) ([]Snapshot, error)
	ListByJobID(ctx context.Context, jobID uuid.UUID) ([]Snapshot, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type pgSnapshotStore struct {
	db *DB
}

// NewSnapshotStore returns a PostgreSQL-backed SnapshotStore.
func NewSnapshotStore(db *DB) SnapshotStore {
	return &pgSnapshotStore{db: db}
}

func (s *pgSnapshotStore) Create(ctx context.Context, snap *Snapshot) error {
	row := s.db.pool.QueryRow(ctx, `
		INSERT INTO snapshots
		  (job_id, engine_snapshot_id, repository_id, hostname, paths, checksum_status)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at`,
		pgtype.UUID{Bytes: snap.JobID, Valid: true},
		snap.EngineSnapshotID,
		pgtype.UUID{Bytes: snap.RepositoryID, Valid: true},
		snap.Hostname,
		orEmptySlice(snap.Paths),
		snap.ChecksumStatus,
	)
	var rawID pgtype.UUID
	if err := row.Scan(&rawID, &snap.CreatedAt); err != nil {
		return err
	}
	snap.ID = uuid.UUID(rawID.Bytes)
	return nil
}

func (s *pgSnapshotStore) GetByID(ctx context.Context, id uuid.UUID) (*Snapshot, error) {
	row := s.db.pool.QueryRow(ctx, `
		SELECT id, job_id, engine_snapshot_id, repository_id, created_at,
		       hostname, paths, checksum_status
		FROM snapshots WHERE id = $1`,
		pgtype.UUID{Bytes: id, Valid: true},
	)
	snap, err := scanSnapshot(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return snap, err
}

func (s *pgSnapshotStore) List(ctx context.Context) ([]Snapshot, error) {
	return s.querySnapshots(ctx, `
		SELECT id, job_id, engine_snapshot_id, repository_id, created_at,
		       hostname, paths, checksum_status
		FROM snapshots ORDER BY created_at DESC`)
}

func (s *pgSnapshotStore) ListByJobID(ctx context.Context, jobID uuid.UUID) ([]Snapshot, error) {
	return s.querySnapshots(ctx, `
		SELECT id, job_id, engine_snapshot_id, repository_id, created_at,
		       hostname, paths, checksum_status
		FROM snapshots WHERE job_id = $1 ORDER BY created_at DESC`,
		pgtype.UUID{Bytes: jobID, Valid: true},
	)
}

func (s *pgSnapshotStore) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := s.db.pool.Exec(ctx, `DELETE FROM snapshots WHERE id = $1`,
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

func (s *pgSnapshotStore) querySnapshots(ctx context.Context, query string, args ...any) ([]Snapshot, error) {
	rows, err := s.db.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var snaps []Snapshot
	for rows.Next() {
		snap, err := scanSnapshot(rows)
		if err != nil {
			return nil, err
		}
		snaps = append(snaps, *snap)
	}
	return snaps, rows.Err()
}

func scanSnapshot(row rowScanner) (*Snapshot, error) {
	var (
		snap    Snapshot
		rawID   pgtype.UUID
		rawJob  pgtype.UUID
		rawRepo pgtype.UUID
	)
	if err := row.Scan(
		&rawID, &rawJob, &snap.EngineSnapshotID, &rawRepo,
		&snap.CreatedAt, &snap.Hostname, &snap.Paths, &snap.ChecksumStatus,
	); err != nil {
		return nil, err
	}
	snap.ID = uuid.UUID(rawID.Bytes)
	snap.JobID = uuid.UUID(rawJob.Bytes)
	snap.RepositoryID = uuid.UUID(rawRepo.Bytes)
	return &snap, nil
}
