ALTER TABLE systems
    DROP COLUMN IF EXISTS data_owner,
    DROP COLUMN IF EXISTS retention_days,
    DROP COLUMN IF EXISTS gdpr_note;
