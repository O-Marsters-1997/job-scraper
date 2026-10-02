-- +goose Up
ALTER TABLE source_targets ADD COLUMN disabled_reason TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE source_targets DROP COLUMN disabled_reason;
