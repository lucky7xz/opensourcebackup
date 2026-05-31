-- B_AUD: Add actor_type and severity to audit_log.
--
-- actor_type: who triggered the event (admin | agent | scheduler | system)
-- severity:   operational classification (info | warning | critical)
--
-- These columns make the audit log queryable by security-relevant dimensions
-- without parsing the free-text `details` field.

ALTER TABLE audit_log
    ADD COLUMN IF NOT EXISTS actor_type TEXT NOT NULL DEFAULT 'system',
    ADD COLUMN IF NOT EXISTS severity   TEXT NOT NULL DEFAULT 'info';

COMMENT ON COLUMN audit_log.actor_type IS
    'Who triggered the event: admin | agent | scheduler | system';
COMMENT ON COLUMN audit_log.severity IS
    'Operational severity: info | warning | critical';

CREATE INDEX IF NOT EXISTS audit_log_severity_idx  ON audit_log (severity, timestamp DESC);
CREATE INDEX IF NOT EXISTS audit_log_action_idx    ON audit_log (action, timestamp DESC);
