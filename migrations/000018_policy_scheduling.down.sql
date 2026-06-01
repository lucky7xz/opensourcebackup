ALTER TABLE backup_policies
    DROP COLUMN IF EXISTS timezone,
    DROP COLUMN IF EXISTS backup_window_start,
    DROP COLUMN IF EXISTS backup_window_end,
    DROP COLUMN IF EXISTS restore_test_schedule,
    DROP COLUMN IF EXISTS retention_schedule,
    DROP COLUMN IF EXISTS if_missed;
