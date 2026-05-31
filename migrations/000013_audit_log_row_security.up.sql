-- ============================================================
-- Audit-Log: Row-Level Security (RLS)
-- Schützt die audit_log-Tabelle auf Datenbankebene:
--   - App-User (opensourcebackup) darf nur INSERT + SELECT
--   - UPDATE und DELETE sind für den App-User verboten
--   - Nur ein DB-Superuser oder ein dedizierter Purge-User
--     kann Zeilen löschen (z.B. für gesetzliche Datenlöschung)
--
-- Damit ist die Unveränderlichkeit des Audit-Logs auch dann
-- gewährleistet, wenn der Anwendungscode kompromittiert wird.
-- ============================================================

-- 1. RLS auf der Tabelle aktivieren
ALTER TABLE audit_log ENABLE ROW LEVEL SECURITY;

-- 2. Sicherstellen, dass der Table-Owner (Superuser) keine RLS-Beschränkung hat
--    (Row Security gilt nicht für den Table-Owner by default — explizit dokumentiert)
ALTER TABLE audit_log FORCE ROW LEVEL SECURITY;

-- 3. Policy: App-User darf alle Zeilen lesen
CREATE POLICY audit_log_select
    ON audit_log
    FOR SELECT
    TO opensourcebackup
    USING (true);

-- 4. Policy: App-User darf neue Zeilen einfügen
CREATE POLICY audit_log_insert
    ON audit_log
    FOR INSERT
    TO opensourcebackup
    WITH CHECK (true);

-- 5. Kein UPDATE/DELETE für App-User — keine Policy = kein Zugriff
--    (PostgreSQL-Default: ohne Policy ist der Zugriff verboten)

-- 6. Kommentar zur Dokumentation
COMMENT ON TABLE audit_log IS
    'Append-only audit trail. RLS enforced: app user (opensourcebackup) '
    'may only INSERT and SELECT. UPDATE/DELETE require superuser or '
    'dedicated purge role. See docs/security/audit-log.md.';
