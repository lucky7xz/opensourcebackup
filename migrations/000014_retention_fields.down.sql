ALTER TABLE backup_policies
    DROP COLUMN IF EXISTS keep_last,
    DROP COLUMN IF EXISTS keep_daily,
    DROP COLUMN IF EXISTS keep_weekly,
    DROP COLUMN IF EXISTS keep_monthly,
    DROP COLUMN IF EXISTS keep_yearly;

DROP INDEX  IF EXISTS idx_backup_jobs_type;
ALTER TABLE backup_jobs DROP COLUMN IF EXISTS type;
