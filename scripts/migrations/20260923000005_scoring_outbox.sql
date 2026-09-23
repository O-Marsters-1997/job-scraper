-- +goose Up
ALTER TABLE job_scores ADD COLUMN score_fingerprint TEXT;
ALTER TABLE job_scores ADD COLUMN score_config_version TIMESTAMPTZ;
ALTER TABLE job_scores ADD COLUMN score_model TEXT;
CREATE TABLE effect_outbox (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id UUID NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    fingerprint TEXT NOT NULL,
    config_version TIMESTAMPTZ NOT NULL,
    model TEXT NOT NULL,
    first_discovery BOOLEAN NOT NULL DEFAULT FALSE,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'running', 'done', 'failed')),
    attempts INT NOT NULL DEFAULT 0,
    due_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    lease_until TIMESTAMPTZ,
    last_error TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (job_id, user_id, fingerprint, config_version, model)
);
CREATE INDEX effect_outbox_due_idx ON effect_outbox (due_at, id) WHERE status IN ('pending', 'running');

-- +goose Down
DROP TABLE effect_outbox;
ALTER TABLE job_scores DROP COLUMN score_model;
ALTER TABLE job_scores DROP COLUMN score_config_version;
ALTER TABLE job_scores DROP COLUMN score_fingerprint;
