-- +goose Up
ALTER TABLE achievements ADD COLUMN context TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE achievements DROP COLUMN context;
