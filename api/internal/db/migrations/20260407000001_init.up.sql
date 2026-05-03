CREATE TABLE users (
    id          BIGSERIAL PRIMARY KEY,
    telegram_id BIGINT      NOT NULL UNIQUE,
    name        TEXT        NOT NULL,
    username    TEXT        NOT NULL,
    photo_url   TEXT        NOT NULL DEFAULT '',
    roles       TEXT[]      NOT NULL DEFAULT '{}',
    is_blocked  BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE sessions (
    id            BIGSERIAL PRIMARY KEY,
    user_id       BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at    TIMESTAMPTZ,
    user_agent    TEXT        NOT NULL,
    ip            TEXT        NOT NULL,
    last_activity TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    status        TEXT        NOT NULL DEFAULT 'active'
                      CHECK (status IN ('active', 'expired', 'inactive'))
);

CREATE INDEX idx_sessions_user_id ON sessions(user_id);
CREATE INDEX idx_sessions_status  ON sessions(status);

CREATE TABLE seasons (
    id         BIGSERIAL PRIMARY KEY,
    title      TEXT        NOT NULL,
    start_date TIMESTAMPTZ NOT NULL,
    end_date   TIMESTAMPTZ NOT NULL,
    is_active  BOOLEAN     NOT NULL DEFAULT FALSE
);

CREATE TABLE season_events (
    id          BIGSERIAL PRIMARY KEY,
    season_id BIGINT      NOT NULL REFERENCES seasons(id) ON DELETE CASCADE,
    title       TEXT        NOT NULL,
    description TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    event_start TIMESTAMPTZ NOT NULL,
    event_end   TIMESTAMPTZ NOT NULL,
    coins_reward INT
);

CREATE INDEX idx_season_events_season_id ON season_events(season_id);

CREATE TABLE season_members (
    id          BIGSERIAL PRIMARY KEY,
    user_id     BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    season_id BIGINT      NOT NULL REFERENCES seasons(id) ON DELETE CASCADE,
    balance     INT         NOT NULL DEFAULT 0,
    total_earned INT        NOT NULL DEFAULT 0,
    updated_at  TIMESTAMPTZ NOT NULL,
    UNIQUE (user_id, season_id)
);

CREATE INDEX idx_season_members_user_id     ON season_members(user_id);
CREATE INDEX idx_season_members_season_id ON season_members(season_id);

-- The sign of `amount` encodes direction: positive = accrual, negative = deduction.
-- ref_id is a nullable FK to the related entity (task, reward, etc.).
CREATE TABLE transactions (
    id         BIGSERIAL PRIMARY KEY,
    member_id BIGINT      NOT NULL REFERENCES season_members(id) ON DELETE CASCADE,
    amount     INT         NOT NULL,
    reason     TEXT        NOT NULL
                   CHECK (reason IN ('event', 'task', 'manual', 'reward')),
    ref_id     BIGINT,
    ref_title  TEXT        NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_transactions_member_id ON transactions(member_id);

CREATE TABLE tasks (
    id           BIGSERIAL PRIMARY KEY,
    season_id  BIGINT  NOT NULL REFERENCES seasons(id) ON DELETE CASCADE,
    category     TEXT    NOT NULL,
    title        TEXT    NOT NULL,
    description  TEXT    NOT NULL DEFAULT '',
    reward_coins INT     NOT NULL DEFAULT 0,
    sort_order   INT     NOT NULL DEFAULT 0,
    is_active    BOOLEAN NOT NULL DEFAULT TRUE,
    metadata     JSONB   NOT NULL DEFAULT '{}'
);

CREATE INDEX idx_tasks_season_id ON tasks(season_id);

CREATE TABLE task_submissions (
    id             BIGSERIAL PRIMARY KEY,
    user_id        BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    task_id        BIGINT      NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    status         TEXT        NOT NULL DEFAULT 'pending'
                       CHECK (status IN ('pending', 'approved', 'rejected')),
    review_comment TEXT        NOT NULL DEFAULT '',
    reviewer_id    BIGINT      REFERENCES users(id) ON DELETE SET NULL,
    submitted_at   TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    reviewed_at    TIMESTAMPTZ
);

CREATE INDEX idx_task_submissions_user_id ON task_submissions(user_id);
CREATE INDEX idx_task_submissions_task_id ON task_submissions(task_id);
CREATE INDEX idx_task_submissions_status  ON task_submissions(status);

CREATE TABLE rewards (
    id          BIGSERIAL PRIMARY KEY,
    season_id BIGINT  NOT NULL REFERENCES seasons(id) ON DELETE CASCADE,
    title       TEXT    NOT NULL,
    description TEXT    NOT NULL DEFAULT '',
    cost_coins  INT     NOT NULL DEFAULT 0,
    "limit"     INT,
    status      TEXT    NOT NULL DEFAULT 'available'
                    CHECK (status IN ('available', 'hidden'))
);

CREATE INDEX idx_rewards_season_id ON rewards(season_id);

CREATE TABLE reward_claims (
    id          BIGSERIAL PRIMARY KEY,
    user_id     BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    reward_id   BIGINT      NOT NULL REFERENCES rewards(id) ON DELETE CASCADE,
    status      TEXT        NOT NULL DEFAULT 'pending'
                    CHECK (status IN ('pending', 'completed', 'cancelled')),
    spent_coins INT         NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_reward_claims_user_id   ON reward_claims(user_id);
CREATE INDEX idx_reward_claims_reward_id ON reward_claims(reward_id);
