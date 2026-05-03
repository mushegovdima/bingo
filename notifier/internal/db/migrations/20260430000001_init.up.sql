CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE notifications (
    id           UUID        PRIMARY KEY,
    type         TEXT        NOT NULL,
    user_id      BIGINT      NOT NULL,
    telegram_id  BIGINT      NOT NULL,
    text         TEXT        NOT NULL,
    send_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    reserved_at  TIMESTAMPTZ,
    processed_at TIMESTAMPTZ
);

-- picks up rows that are ready, not yet processed, and not currently reserved
CREATE INDEX idx_notifications_claimable ON notifications (send_at)
    WHERE processed_at IS NULL;
