ALTER TABLE backup_jobs
    DROP COLUMN IF EXISTS progress_phase,
    DROP COLUMN IF EXISTS progress_percent,
    DROP COLUMN IF EXISTS progress_bytes_done,
    DROP COLUMN IF EXISTS progress_bytes_total,
    DROP COLUMN IF EXISTS progress_files_done,
    DROP COLUMN IF EXISTS progress_files_total,
    DROP COLUMN IF EXISTS progress_throughput_bps,
    DROP COLUMN IF EXISTS last_progress_at;
