ALTER TABLE backup_jobs
    DROP COLUMN IF EXISTS cancel_requested_at,
    DROP COLUMN IF EXISTS cancel_reason;
