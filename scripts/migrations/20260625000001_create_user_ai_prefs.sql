-- +goose Up
CREATE TABLE user_ai_prefs (
    id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id           UUID        UNIQUE NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    suitability_model TEXT        NOT NULL DEFAULT 'claude-haiku-4-5-20251001',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE user_ai_prefs;
