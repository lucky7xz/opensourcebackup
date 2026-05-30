CREATE TABLE systems (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    hostname     TEXT        NOT NULL,
    os           TEXT,
    agent_version TEXT,
    last_seen    TIMESTAMPTZ,
    tags         JSONB       NOT NULL DEFAULT '{}',
    risk_class   TEXT        NOT NULL DEFAULT 'standard',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_systems_hostname ON systems (hostname);
CREATE INDEX idx_systems_risk_class ON systems (risk_class);
