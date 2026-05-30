CREATE TABLE repositories (
    id                   UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    type                 TEXT        NOT NULL,
    location             TEXT        NOT NULL,
    encryption_mode      TEXT,
    object_lock_enabled  BOOLEAN     NOT NULL DEFAULT false,
    retention_policy_id  UUID,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
