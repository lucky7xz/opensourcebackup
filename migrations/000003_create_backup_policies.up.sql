CREATE TABLE backup_policies (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name       TEXT        NOT NULL,
    includes   TEXT[]      NOT NULL DEFAULT '{}',
    excludes   TEXT[]      NOT NULL DEFAULT '{}',
    schedule   TEXT,
    retention  JSONB       NOT NULL DEFAULT '{}',
    engine     TEXT        NOT NULL,
    pre_hooks  TEXT[]      NOT NULL DEFAULT '{}',
    post_hooks TEXT[]      NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
