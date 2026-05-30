CREATE TABLE snapshots (
    id                 UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id             UUID        NOT NULL REFERENCES backup_jobs(id),
    engine_snapshot_id TEXT        NOT NULL,
    repository_id      UUID        NOT NULL REFERENCES repositories(id),
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    hostname           TEXT,
    paths              TEXT[]      NOT NULL DEFAULT '{}',
    checksum_status    TEXT        NOT NULL DEFAULT 'unverified'
);

CREATE INDEX idx_snapshots_job_id        ON snapshots (job_id);
CREATE INDEX idx_snapshots_repository_id ON snapshots (repository_id);
CREATE INDEX idx_snapshots_created_at    ON snapshots (created_at DESC);
