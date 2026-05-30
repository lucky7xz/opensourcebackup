CREATE TABLE backup_jobs (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    system_id     UUID        NOT NULL REFERENCES systems(id),
    policy_id     UUID        NOT NULL REFERENCES backup_policies(id),
    started_at    TIMESTAMPTZ,
    finished_at   TIMESTAMPTZ,
    status        TEXT        NOT NULL DEFAULT 'running',
    bytes_scanned BIGINT,
    bytes_uploaded BIGINT,
    error_summary TEXT,
    raw_output    JSONB,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_backup_jobs_system_id ON backup_jobs (system_id);
CREATE INDEX idx_backup_jobs_policy_id ON backup_jobs (policy_id);
CREATE INDEX idx_backup_jobs_status    ON backup_jobs (status);
CREATE INDEX idx_backup_jobs_started_at ON backup_jobs (started_at DESC);
