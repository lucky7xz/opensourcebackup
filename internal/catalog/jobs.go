package catalog

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

const jobSelect = `
	SELECT id, system_id, policy_id, type, started_at, finished_at, status,
	       bytes_scanned, bytes_uploaded, error_summary, raw_output, created_at,
	       progress_phase, progress_percent, progress_bytes_done, progress_bytes_total,
	       progress_files_done, progress_files_total, progress_throughput_bps, last_progress_at
	FROM backup_jobs`

// JobStore defines data access for the backup_jobs table.
type JobStore interface {
	Create(ctx context.Context, j *BackupJob) error
	GetByID(ctx context.Context, id uuid.UUID) (*BackupJob, error)
	List(ctx context.Context) ([]BackupJob, error)
	ListBySystemID(ctx context.Context, systemID uuid.UUID) ([]BackupJob, error)
	// ListPendingBySystemID returns pending backup jobs (type="backup") for the agent.
	ListPendingBySystemID(ctx context.Context, systemID uuid.UUID) ([]BackupJob, error)
	// ListPendingRetentionBySystemID returns pending retention jobs for the agent.
	ListPendingRetentionBySystemID(ctx context.Context, systemID uuid.UUID) ([]BackupJob, error)
	LatestByPolicyID(ctx context.Context, policyID uuid.UUID) (*BackupJob, error)
	Update(ctx context.Context, j *BackupJob) error
	// UpdateProgress writes a live progress snapshot (lightweight, called every few
	// seconds while a backup runs). It touches only the progress_* columns.
	UpdateProgress(ctx context.Context, id uuid.UUID, p JobProgress) error
	// FinalizeProgress pins a completed job to 100% so a successful job never
	// lingers at e.g. 98.7%. Leaves progress untouched for failed/cancelled jobs.
	FinalizeProgress(ctx context.Context, id uuid.UUID) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type pgJobStore struct {
	db *DB
}

// NewJobStore returns a PostgreSQL-backed JobStore.
func NewJobStore(db *DB) JobStore {
	return &pgJobStore{db: db}
}

func (s *pgJobStore) Create(ctx context.Context, j *BackupJob) error {
	rawOutput, err := marshalJSONB(j.RawOutput)
	if err != nil {
		return err
	}
	jobType := j.Type
	if jobType == "" {
		jobType = JobTypeBackup
	}
	row := s.db.pool.QueryRow(ctx, `
		INSERT INTO backup_jobs
		  (system_id, policy_id, type, started_at, finished_at, status,
		   bytes_scanned, bytes_uploaded, error_summary, raw_output)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		RETURNING id, created_at`,
		pgtype.UUID{Bytes: j.SystemID, Valid: true},
		pgtype.UUID{Bytes: j.PolicyID, Valid: true},
		jobType,
		j.StartedAt, j.FinishedAt, j.Status,
		j.BytesScanned, j.BytesUploaded, j.ErrorSummary, rawOutput,
	)
	var rawID pgtype.UUID
	if err := row.Scan(&rawID, &j.CreatedAt); err != nil {
		return err
	}
	j.ID = uuid.UUID(rawID.Bytes)
	j.Type = jobType
	return nil
}

func (s *pgJobStore) GetByID(ctx context.Context, id uuid.UUID) (*BackupJob, error) {
	row := s.db.pool.QueryRow(ctx,
		jobSelect+` WHERE id = $1`,
		pgtype.UUID{Bytes: id, Valid: true},
	)
	j, err := scanJob(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return j, err
}

func (s *pgJobStore) List(ctx context.Context) ([]BackupJob, error) {
	return s.queryJobs(ctx, jobSelect+` ORDER BY created_at DESC`)
}

func (s *pgJobStore) ListBySystemID(ctx context.Context, systemID uuid.UUID) ([]BackupJob, error) {
	return s.queryJobs(ctx,
		jobSelect+` WHERE system_id = $1 ORDER BY created_at DESC`,
		pgtype.UUID{Bytes: systemID, Valid: true},
	)
}

func (s *pgJobStore) ListPendingBySystemID(ctx context.Context, systemID uuid.UUID) ([]BackupJob, error) {
	return s.queryJobs(ctx,
		jobSelect+` WHERE system_id = $1 AND status = 'pending' AND type = $2 ORDER BY created_at ASC`,
		pgtype.UUID{Bytes: systemID, Valid: true}, JobTypeBackup,
	)
}

func (s *pgJobStore) ListPendingRetentionBySystemID(ctx context.Context, systemID uuid.UUID) ([]BackupJob, error) {
	return s.queryJobs(ctx,
		jobSelect+` WHERE system_id = $1 AND status = 'pending' AND type = $2 ORDER BY created_at ASC`,
		pgtype.UUID{Bytes: systemID, Valid: true}, JobTypeRetention,
	)
}

func (s *pgJobStore) LatestByPolicyID(ctx context.Context, policyID uuid.UUID) (*BackupJob, error) {
	row := s.db.pool.QueryRow(ctx,
		jobSelect+` WHERE policy_id = $1 ORDER BY created_at DESC LIMIT 1`,
		pgtype.UUID{Bytes: policyID, Valid: true},
	)
	j, err := scanJob(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return j, err
}

func (s *pgJobStore) Update(ctx context.Context, j *BackupJob) error {
	rawOutput, err := marshalJSONB(j.RawOutput)
	if err != nil {
		return err
	}
	tag, err := s.db.pool.Exec(ctx, `
		UPDATE backup_jobs
		SET started_at=$1, finished_at=$2, status=$3,
		    bytes_scanned=$4, bytes_uploaded=$5, error_summary=$6, raw_output=$7
		WHERE id=$8`,
		j.StartedAt, j.FinishedAt, j.Status,
		j.BytesScanned, j.BytesUploaded, j.ErrorSummary, rawOutput,
		pgtype.UUID{Bytes: j.ID, Valid: true},
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *pgJobStore) UpdateProgress(ctx context.Context, id uuid.UUID, p JobProgress) error {
	tag, err := s.db.pool.Exec(ctx, `
		UPDATE backup_jobs
		SET progress_phase=$1, progress_percent=$2,
		    progress_bytes_done=$3, progress_bytes_total=$4,
		    progress_files_done=$5, progress_files_total=$6,
		    progress_throughput_bps=$7, last_progress_at=NOW()
		WHERE id=$8`,
		p.Phase, p.Percent, p.BytesDone, p.BytesTotal,
		p.FilesDone, p.FilesTotal, p.ThroughputBps,
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

func (s *pgJobStore) FinalizeProgress(ctx context.Context, id uuid.UUID) error {
	tag, err := s.db.pool.Exec(ctx, `
		UPDATE backup_jobs
		SET progress_percent=100, last_progress_at=NOW()
		WHERE id=$1`,
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

func (s *pgJobStore) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := s.db.pool.Exec(ctx, `DELETE FROM backup_jobs WHERE id = $1`,
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

func (s *pgJobStore) queryJobs(ctx context.Context, query string, args ...any) ([]BackupJob, error) {
	rows, err := s.db.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []BackupJob
	for rows.Next() {
		j, err := scanJob(rows)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, *j)
	}
	return jobs, rows.Err()
}

func scanJob(row rowScanner) (*BackupJob, error) {
	var (
		j         BackupJob
		rawID     pgtype.UUID
		rawSysID  pgtype.UUID
		rawPolID  pgtype.UUID
		rawOutput []byte
	)
	if err := row.Scan(
		&rawID, &rawSysID, &rawPolID, &j.Type,
		&j.StartedAt, &j.FinishedAt, &j.Status,
		&j.BytesScanned, &j.BytesUploaded, &j.ErrorSummary, &rawOutput,
		&j.CreatedAt,
		&j.ProgressPhase, &j.ProgressPercent, &j.ProgressBytesDone, &j.ProgressBytesTotal,
		&j.ProgressFilesDone, &j.ProgressFilesTotal, &j.ProgressThroughputBps, &j.LastProgressAt,
	); err != nil {
		return nil, err
	}
	j.ID = uuid.UUID(rawID.Bytes)
	j.SystemID = uuid.UUID(rawSysID.Bytes)
	j.PolicyID = uuid.UUID(rawPolID.Bytes)
	if j.Type == "" {
		j.Type = JobTypeBackup
	}
	if len(rawOutput) > 0 {
		if err := json.Unmarshal(rawOutput, &j.RawOutput); err != nil {
			return nil, err
		}
	}
	return &j, nil
}
