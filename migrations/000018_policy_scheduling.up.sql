-- B_SCHED_UI: Advanced policy scheduling fields.
--
-- Existing: schedule (cron string), retention (jsonb), keep_* columns
-- New: timezone, backup window, restore test schedule, retention schedule, if_missed behaviour

ALTER TABLE backup_policies
    ADD COLUMN IF NOT EXISTS timezone            TEXT NOT NULL DEFAULT 'UTC',
    ADD COLUMN IF NOT EXISTS backup_window_start TEXT NOT NULL DEFAULT '',   -- e.g. "22:00"
    ADD COLUMN IF NOT EXISTS backup_window_end   TEXT NOT NULL DEFAULT '',   -- e.g. "06:00"
    ADD COLUMN IF NOT EXISTS restore_test_schedule TEXT NOT NULL DEFAULT '', -- cron or ''
    ADD COLUMN IF NOT EXISTS retention_schedule  TEXT NOT NULL DEFAULT '',   -- cron or ''
    ADD COLUMN IF NOT EXISTS if_missed           TEXT NOT NULL DEFAULT 'run_asap'; -- run_asap | skip

COMMENT ON COLUMN backup_policies.timezone             IS 'IANA timezone for schedule evaluation (e.g. Europe/Berlin)';
COMMENT ON COLUMN backup_policies.backup_window_start  IS 'Backups only allowed after this time (HH:MM), empty = no restriction';
COMMENT ON COLUMN backup_policies.backup_window_end    IS 'Backups only allowed before this time (HH:MM), empty = no restriction';
COMMENT ON COLUMN backup_policies.restore_test_schedule IS 'Cron for automatic restore tests, empty = manual only';
COMMENT ON COLUMN backup_policies.retention_schedule   IS 'Cron for prune/retention run, empty = manual only';
COMMENT ON COLUMN backup_policies.if_missed            IS 'run_asap: run immediately when window opens; skip: wait for next schedule';
