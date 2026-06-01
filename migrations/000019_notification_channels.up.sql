-- Notification channels for alerts (webhook, email)
CREATE TABLE notification_channels (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name       TEXT        NOT NULL,
    type       TEXT        NOT NULL,  -- 'webhook' | 'email'
    url        TEXT        NOT NULL DEFAULT '',  -- webhook URL or SMTP server
    target     TEXT        NOT NULL DEFAULT '',  -- email address or webhook target
    enabled    BOOLEAN     NOT NULL DEFAULT true,
    min_severity TEXT      NOT NULL DEFAULT 'warning', -- 'info' | 'warning' | 'critical'
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE notification_channels IS 'Outbound notification targets for health alerts';
