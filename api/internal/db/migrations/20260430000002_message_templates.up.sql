CREATE TABLE message_templates (
    id          BIGSERIAL    PRIMARY KEY,
    codename    TEXT         NOT NULL UNIQUE,
    body        TEXT         NOT NULL,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- History records body changes only.
CREATE TABLE message_template_history (
    id          BIGSERIAL    PRIMARY KEY,
    template_id BIGINT       NOT NULL REFERENCES message_templates(id) ON DELETE CASCADE,
    body        TEXT         NOT NULL,
    changed_by  BIGINT       NOT NULL REFERENCES users(id),
    changed_at  TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_message_template_history_template_id ON message_template_history(template_id);

-- Seed: initial notification templates.
-- Body syntax: {{argKey}} placeholders, HTML for Telegram.
-- Accepted keys per template are defined in internal/notifications/notifications.go.

INSERT INTO message_templates (codename, body) VALUES
(
    'season_available',
    $$🚀 <b>Новый сезон открыт!</b>

Сезон <b>«{{Title}}»</b> уже доступен.
📅 {{StartDate}} — {{EndDate}}

Заходи, выполняй задачи и зарабатывай монеты!$$
),
(
    'task_approved',
    $$🏆 <b>Задача выполнена!</b>

Твоё выполнение задачи <b>«{{TaskTitle}}»</b> подтверждено менеджером.
Начислено монет: <b>{{Coins}}</b> 🪙$$
),
(
    'claim_submitted',
    $$🎁 <b>Заявка на приз принята!</b>

Ты подал заявку на <b>«{{RewardTitle}}»</b>.
Списано монет: <b>{{SpentCoins}}</b>

Ожидай подтверждения от менеджера.$$
),
(
    'claim_completed',
    $$✅ <b>Приз получен!</b>

Твоя заявка на <b>«{{RewardTitle}}»</b> подтверждена.
Приз выдан — поздравляем! 🎉$$
),
(
    'claim_cancelled',
    $$❌ <b>Заявка отменена</b>

Заявка на <b>«{{RewardTitle}}»</b> была отменена.
Возвращено монет: <b>{{RefundedCoins}}</b>$$
);
