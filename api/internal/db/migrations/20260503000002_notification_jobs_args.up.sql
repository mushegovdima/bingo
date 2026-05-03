ALTER TABLE notification_jobs
    ADD COLUMN IF NOT EXISTS args JSONB NOT NULL DEFAULT '{}'::jsonb;
