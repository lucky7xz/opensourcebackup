-- B_JOB_CANCEL: cooperative cancellation of a running backup (operational safety).
-- An operator requests cancel (cancel_requested_at); the agent observes it, stops
-- restic via context cancel, and reports the job as 'cancelled' (NOT 'failed').
ALTER TABLE backup_jobs
    ADD COLUMN IF NOT EXISTS cancel_requested_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS cancel_reason       TEXT NOT NULL DEFAULT '';

COMMENT ON COLUMN backup_jobs.cancel_requested_at IS 'Set when an operator requests a stop; NULL otherwise. The agent polls this while a backup runs.';
COMMENT ON COLUMN backup_jobs.cancel_reason       IS 'Operator-supplied reason for the stop (audited).';
