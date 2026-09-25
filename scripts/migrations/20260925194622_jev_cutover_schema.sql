-- +goose Up
ALTER TABLE job_scores
    ADD COLUMN criteria   JSONB   NOT NULL DEFAULT '{}',
    ADD COLUMN confidence REAL,
    ADD COLUMN cost       NUMERIC;

ALTER TABLE job_scores
    DROP COLUMN relevance_score,
    DROP COLUMN reasoning,
    DROP COLUMN matched,
    DROP COLUMN missing,
    DROP COLUMN suitability_skipped;

ALTER TABLE search_config DROP COLUMN suitability_rubric;

DROP TABLE user_ai_prefs;

-- +goose Down
CREATE TABLE user_ai_prefs (
    id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id           UUID        UNIQUE NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    suitability_model TEXT        NOT NULL DEFAULT 'claude-haiku-4-5-20251001',
    reasoning_model   TEXT        NOT NULL DEFAULT 'claude-sonnet-4-6',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE search_config ADD COLUMN suitability_rubric TEXT NOT NULL DEFAULT '';

ALTER TABLE job_scores
    ADD COLUMN relevance_score     INT,
    ADD COLUMN reasoning           TEXT,
    ADD COLUMN matched             TEXT[],
    ADD COLUMN missing             TEXT[],
    ADD COLUMN suitability_skipped BOOLEAN NOT NULL DEFAULT false;

ALTER TABLE job_scores
    DROP COLUMN criteria,
    DROP COLUMN confidence,
    DROP COLUMN cost;
