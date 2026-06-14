-- +goose Up
CREATE TABLE IF NOT EXISTS job_scores (
    id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id            UUID        NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    user_id           UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    relevance_score   INT,
    suitability_score INT,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (job_id, user_id)
);
-- +goose Down
DROP TABLE IF EXISTS job_scores;
