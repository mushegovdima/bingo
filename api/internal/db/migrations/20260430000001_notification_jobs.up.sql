CREATE TABLE notification_jobs (
    id          BIGSERIAL    PRIMARY KEY,
    type        TEXT         NOT NULL,
    text        TEXT         NOT NULL,
    args        JSONB        NOT NULL DEFAULT '{}'::jsonb,
    filter      JSONB        NOT NULL DEFAULT '{}'::jsonb,
    status      TEXT         NOT NULL DEFAULT 'pending'
                    CHECK (status IN ('pending', 'running', 'done', 'failed')),
    cursor      BIGINT       NOT NULL DEFAULT 0,
    attempts    INT          NOT NULL DEFAULT 0,
    last_error  TEXT,
    locked_at   TIMESTAMPTZ,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Partial index used by the worker to claim pending jobs (and reclaim stuck running ones).
CREATE INDEX idx_notification_jobs_claimable
    ON notification_jobs (created_at)
    WHERE status IN ('pending', 'running');
