-- GDPR fields for systems table:
-- data_owner: responsible contact (name or email) — optional, helps with GDPR requests
-- retention_days: how many days backup snapshots are kept before auto-prune
-- gdpr_note: free-text field for data processing notes

ALTER TABLE systems
    ADD COLUMN IF NOT EXISTS data_owner     TEXT    NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS retention_days INTEGER NOT NULL DEFAULT 90,
    ADD COLUMN IF NOT EXISTS gdpr_note      TEXT    NOT NULL DEFAULT '';

COMMENT ON COLUMN systems.data_owner     IS 'Responsible contact for GDPR purposes';
COMMENT ON COLUMN systems.retention_days IS 'Days to retain backup snapshots before auto-prune';
COMMENT ON COLUMN systems.gdpr_note      IS 'Data processing notes for GDPR documentation';
