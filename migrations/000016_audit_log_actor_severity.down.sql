DROP INDEX IF EXISTS audit_log_severity_idx;
DROP INDEX IF EXISTS audit_log_action_idx;
ALTER TABLE audit_log DROP COLUMN IF EXISTS actor_type;
ALTER TABLE audit_log DROP COLUMN IF EXISTS severity;
