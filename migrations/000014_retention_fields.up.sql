-- B_RET: Retention fields for backup policies + job type
--
-- backup_policies: add explicit retention columns alongside the existing JSONB.
-- Using typed columns allows queries like "find policies with keep_last > 0"
-- without parsing JSONB.
--
-- backup_jobs: add type column to distinguish backup vs. retention jobs.
-- Allows the agent to pick up retention jobs separately from backup jobs.

ALTER TABLE backup_policies
    ADD COLUMN IF NOT EXISTS keep_last    INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS keep_daily   INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS keep_weekly  INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS keep_monthly INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS keep_yearly  INTEGER NOT NULL DEFAULT 0;

COMMENT ON COLUMN backup_policies.keep_last    IS 'Keep N most recent snapshots (restic --keep-last)';
COMMENT ON COLUMN backup_policies.keep_daily   IS 'Keep one snapshot per day for N days (restic --keep-daily)';
COMMENT ON COLUMN backup_policies.keep_weekly  IS 'Keep one snapshot per week for N weeks (restic --keep-weekly)';
COMMENT ON COLUMN backup_policies.keep_monthly IS 'Keep one snapshot per month for N months (restic --keep-monthly)';
COMMENT ON COLUMN backup_policies.keep_yearly  IS 'Keep one snapshot per year for N years (restic --keep-yearly)';

ALTER TABLE backup_jobs
    ADD COLUMN IF NOT EXISTS type TEXT NOT NULL DEFAULT 'backup';

COMMENT ON COLUMN backup_jobs.type IS 'Job type: backup | retention';

CREATE INDEX IF NOT EXISTS idx_backup_jobs_type ON backup_jobs (type);
