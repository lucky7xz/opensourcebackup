-- B_RBAC: Users table for role-based access control.
--
-- Three roles:
--   admin    — full access, including destructive operations
--   operator — operational access (no delete repos/policies, no role management)
--   viewer   — read-only access to all data
--
-- Password hashing: bcrypt cost 12 (enforced in application, not DB).
-- Sessions: stored in-memory with user_id reference (no separate sessions table for MVP).
-- Bootstrap: if ADMIN_EMAIL + ADMIN_PASSWORD are set and no admin user exists, one is created.

CREATE TABLE users (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    email         TEXT        NOT NULL UNIQUE,
    password_hash TEXT        NOT NULL,
    role          TEXT        NOT NULL DEFAULT 'viewer',
    display_name  TEXT        NOT NULL DEFAULT '',
    disabled_at   TIMESTAMPTZ,             -- NULL = active; set to disable without deleting
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON COLUMN users.role IS 'admin | operator | viewer';
COMMENT ON COLUMN users.disabled_at IS 'NULL = active. Set to disable login without deleting the user.';

CREATE INDEX users_email_idx ON users (email);
CREATE INDEX users_role_idx  ON users (role);
