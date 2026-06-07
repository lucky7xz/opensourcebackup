-- B_JOB_PROGRESS: live progress for running backups.
-- Aggregate numbers only — NO file paths/names are stored (privacy / data minimisation).
ALTER TABLE backup_jobs
    ADD COLUMN IF NOT EXISTS progress_phase           TEXT             NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS progress_percent         DOUBLE PRECISION NOT NULL DEFAULT 0,  -- 0..100
    ADD COLUMN IF NOT EXISTS progress_bytes_done      BIGINT           NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS progress_bytes_total     BIGINT           NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS progress_files_done      INTEGER          NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS progress_files_total     INTEGER          NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS progress_throughput_bps  BIGINT           NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS last_progress_at         TIMESTAMPTZ;

COMMENT ON COLUMN backup_jobs.progress_percent        IS '0..100, derived from restic percent_done*100';
COMMENT ON COLUMN backup_jobs.progress_throughput_bps IS 'Bytes/s, computed by the agent (restic does not report it)';
COMMENT ON COLUMN backup_jobs.last_progress_at        IS 'Last progress update; NULL until first report. Used to detect stalled jobs.';
