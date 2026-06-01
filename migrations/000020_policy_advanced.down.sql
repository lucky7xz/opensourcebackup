ALTER TABLE backup_policies
    DROP COLUMN IF EXISTS bandwidth_limit_kbps,
    DROP COLUMN IF EXISTS verify_schedule,
    DROP COLUMN IF EXISTS verify_full;
ALTER TABLE systems DROP COLUMN IF EXISTS agent_version_reported;
