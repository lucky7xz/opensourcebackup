-- Advanced policy features: bandwidth throttling + verify schedule + auto-update
ALTER TABLE backup_policies
    ADD COLUMN IF NOT EXISTS bandwidth_limit_kbps INTEGER NOT NULL DEFAULT 0,  -- 0 = unlimited
    ADD COLUMN IF NOT EXISTS verify_schedule       TEXT    NOT NULL DEFAULT '',  -- cron for restic check
    ADD COLUMN IF NOT EXISTS verify_full           BOOLEAN NOT NULL DEFAULT false; -- --read-data flag

COMMENT ON COLUMN backup_policies.bandwidth_limit_kbps IS '0 = unlimited. Upload bandwidth cap in KB/s passed to restic --limit-upload';
COMMENT ON COLUMN backup_policies.verify_schedule      IS 'Cron for automatic restic check (no full restore, just hash verification)';
COMMENT ON COLUMN backup_policies.verify_full          IS 'If true, reads all data from repository (slow but thorough)';

-- Agent auto-update: track current binary version per system
ALTER TABLE systems
    ADD COLUMN IF NOT EXISTS agent_version_reported TEXT NOT NULL DEFAULT '';

COMMENT ON COLUMN systems.agent_version_reported IS 'Agent binary version reported on last heartbeat';
