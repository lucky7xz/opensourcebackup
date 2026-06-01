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
	// ListWithRetention returns policies that have at least one retention rule configured.
	ListWithRetention(ctx context.Context) ([]BackupPolicy, error)
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
	tz := p.ScheduleConfig.Timezone
	if tz == "" {
		tz = "UTC"
	}
	ifMissed := p.ScheduleConfig.IfMissed
	if ifMissed == "" {
		ifMissed = "run_asap"
	}
	row := s.db.pool.QueryRow(ctx, `
		INSERT INTO backup_policies
		  (name, includes, excludes, schedule, retention, engine, pre_hooks, post_hooks, repository_id,
		   keep_last, keep_daily, keep_weekly, keep_monthly, keep_yearly,
		   timezone, backup_window_start, backup_window_end,
		   restore_test_schedule, retention_schedule, if_missed)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20)
		RETURNING id, created_at`,
		p.Name, orEmptySlice(p.Includes), orEmptySlice(p.Excludes), p.Schedule,
		retentionJSON, p.Engine, orEmptySlice(p.PreHooks), orEmptySlice(p.PostHooks),
		uuidPtrToRaw(p.RepositoryID),
		p.RetentionPlan.KeepLast, p.RetentionPlan.KeepDaily,
		p.RetentionPlan.KeepWeekly, p.RetentionPlan.KeepMonthly, p.RetentionPlan.KeepYearly,
		tz, p.ScheduleConfig.WindowStart, p.ScheduleConfig.WindowEnd,
		p.ScheduleConfig.RestoreTestCron, p.ScheduleConfig.RetentionCron, ifMissed,
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
		SELECT id, name, includes, excludes, schedule, retention, engine,
		       pre_hooks, post_hooks, repository_id, created_at,
		       keep_last, keep_daily, keep_weekly, keep_monthly, keep_yearly,
		       timezone, backup_window_start, backup_window_end,
		       restore_test_schedule, retention_schedule, if_missed
		FROM backup_policies WHERE id = $1`,
		pgtype.UUID{Bytes: id, Valid: true},
	)
	p, err := scanPolicy(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return p, err
}

const policySelect = `
	SELECT id, name, includes, excludes, schedule, retention, engine,
	       pre_hooks, post_hooks, repository_id, created_at,
	       keep_last, keep_daily, keep_weekly, keep_monthly, keep_yearly,
	       timezone, backup_window_start, backup_window_end,
	       restore_test_schedule, retention_schedule, if_missed
	FROM backup_policies`

func (s *pgPolicyStore) List(ctx context.Context) ([]BackupPolicy, error) {
	return s.queryPolicies(ctx, policySelect+` ORDER BY created_at DESC`)
}

func (s *pgPolicyStore) ListWithRetention(ctx context.Context) ([]BackupPolicy, error) {
	return s.queryPolicies(ctx, policySelect+`
		WHERE (keep_last > 0 OR keep_daily > 0 OR keep_weekly > 0 OR keep_monthly > 0 OR keep_yearly > 0)
		ORDER BY created_at DESC`)
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

// ── Internal helpers ──────────────────────────────────────────────────────────

func (s *pgPolicyStore) Update(ctx context.Context, p *BackupPolicy) error {
	retentionJSON, err := marshalJSONB(p.Retention)
	if err != nil {
		return err
	}
	tz := p.ScheduleConfig.Timezone
	if tz == "" { tz = "UTC" }
	ifMissed := p.ScheduleConfig.IfMissed
	if ifMissed == "" { ifMissed = "run_asap" }
	tag, err := s.db.pool.Exec(ctx, `
		UPDATE backup_policies
		SET name=$1, includes=$2, excludes=$3, schedule=$4,
		    retention=$5, engine=$6, pre_hooks=$7, post_hooks=$8, repository_id=$9,
		    keep_last=$10, keep_daily=$11, keep_weekly=$12, keep_monthly=$13, keep_yearly=$14,
		    timezone=$15, backup_window_start=$16, backup_window_end=$17,
		    restore_test_schedule=$18, retention_schedule=$19, if_missed=$20
		WHERE id=$21`,
		p.Name, orEmptySlice(p.Includes), orEmptySlice(p.Excludes), p.Schedule,
		retentionJSON, p.Engine, orEmptySlice(p.PreHooks), orEmptySlice(p.PostHooks),
		uuidPtrToRaw(p.RepositoryID),
		p.RetentionPlan.KeepLast, p.RetentionPlan.KeepDaily,
		p.RetentionPlan.KeepWeekly, p.RetentionPlan.KeepMonthly, p.RetentionPlan.KeepYearly,
		tz, p.ScheduleConfig.WindowStart, p.ScheduleConfig.WindowEnd,
		p.ScheduleConfig.RestoreTestCron, p.ScheduleConfig.RetentionCron, ifMissed,
		pgtype.UUID{Bytes: p.ID, Valid: true},
	)
	if err != nil { return err }
	if tag.RowsAffected() == 0 { return ErrNotFound }
	return nil
}

func (s *pgPolicyStore) queryPolicies(ctx context.Context, query string, args ...any) ([]BackupPolicy, error) {
	rows, err := s.db.pool.Query(ctx, query, args...)
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
		&p.RetentionPlan.KeepLast, &p.RetentionPlan.KeepDaily,
		&p.RetentionPlan.KeepWeekly, &p.RetentionPlan.KeepMonthly, &p.RetentionPlan.KeepYearly,
		&p.ScheduleConfig.Timezone, &p.ScheduleConfig.WindowStart, &p.ScheduleConfig.WindowEnd,
		&p.ScheduleConfig.RestoreTestCron, &p.ScheduleConfig.RetentionCron, &p.ScheduleConfig.IfMissed,
	); err != nil {
		return nil, err
	}
	p.ID = uuid.UUID(rawID.Bytes)
	// Sync legacy Schedule field into ScheduleConfig
	if p.Schedule != nil && p.ScheduleConfig.Cron == "" {
		p.ScheduleConfig.Cron = *p.Schedule
	}
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
