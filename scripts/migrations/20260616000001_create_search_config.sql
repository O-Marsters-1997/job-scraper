-- +goose Up
CREATE TABLE IF NOT EXISTS search_config (
    id                 UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id            UUID        NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    role               TEXT        NOT NULL DEFAULT '',
    location           TEXT        NOT NULL DEFAULT '',
    keywords           TEXT[]      NOT NULL DEFAULT '{}',
    suitability_rubric TEXT        NOT NULL DEFAULT '',
    relevance_cutoff   INT         NOT NULL DEFAULT 0,
    notify_threshold   INT         NOT NULL DEFAULT 70,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
-- +goose Down
DROP TABLE IF EXISTS search_config;
