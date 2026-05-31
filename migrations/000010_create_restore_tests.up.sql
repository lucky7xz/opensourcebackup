CREATE TABLE restore_tests (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    snapshot_id     UUID        NOT NULL REFERENCES snapshots(id) ON DELETE CASCADE,
    system_id       UUID        NOT NULL REFERENCES systems(id)   ON DELETE CASCADE,
    repository_id   UUID        NOT NULL REFERENCES repositories(id),
    status          TEXT        NOT NULL DEFAULT 'pending',
    target_path     TEXT,
    started_at      TIMESTAMPTZ,
    finished_at     TIMESTAMPTZ,
    verified_files  INTEGER,
    verified_bytes  BIGINT,
    error_summary   TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_restore_tests_snapshot_id   ON restore_tests (snapshot_id);
CREATE INDEX idx_restore_tests_system_id     ON restore_tests (system_id);
CREATE INDEX idx_restore_tests_status        ON restore_tests (status);
