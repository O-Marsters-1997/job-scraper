-- +goose Up
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS work_arrangement TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE jobs DROP COLUMN IF EXISTS work_arrangement;
