CREATE TABLE agent_enrollment_tokens (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    system_id  UUID        NOT NULL REFERENCES systems(id),
    token_hash TEXT        NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at    TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_enrollment_tokens_system_id ON agent_enrollment_tokens (system_id);
