-- Audit log — append-only table for GDPR compliance and security monitoring.
-- Rows are NEVER updated. Only a privileged GDPR purge operation may delete rows
-- (and must write a gdpr_purge entry before doing so).

CREATE TABLE audit_log (
    id            BIGSERIAL    PRIMARY KEY,
    timestamp     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    action        TEXT         NOT NULL,
    resource_type TEXT         NOT NULL DEFAULT '',
    resource_id   TEXT         NOT NULL DEFAULT '',
    actor         TEXT         NOT NULL DEFAULT '',  -- 'admin', 'agent:<system_id>', 'system'
    ip            TEXT         NOT NULL DEFAULT '',
    user_agent    TEXT         NOT NULL DEFAULT '',
    details       TEXT         NOT NULL DEFAULT '',
    success       BOOLEAN      NOT NULL DEFAULT TRUE
);

-- Index for per-resource lookups (GDPR export, incident investigation)
CREATE INDEX audit_log_resource_idx ON audit_log (resource_type, resource_id, timestamp DESC);
-- Index for actor-based lookups
CREATE INDEX audit_log_actor_idx    ON audit_log (actor, timestamp DESC);
-- Index for time-range queries
CREATE INDEX audit_log_time_idx     ON audit_log (timestamp DESC);

COMMENT ON TABLE audit_log IS
    'Immutable security and GDPR audit trail. Never UPDATE or DELETE rows except via controlled purge.';
