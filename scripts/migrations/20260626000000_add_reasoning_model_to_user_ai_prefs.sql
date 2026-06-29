-- +goose Up
ALTER TABLE user_ai_prefs ADD COLUMN reasoning_model TEXT NOT NULL DEFAULT 'claude-sonnet-4-6';

-- +goose Down
ALTER TABLE user_ai_prefs DROP COLUMN reasoning_model;
