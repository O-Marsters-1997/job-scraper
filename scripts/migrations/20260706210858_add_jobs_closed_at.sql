-- +goose Up
ALTER TABLE jobs ADD COLUMN closed_at TIMESTAMPTZ;

-- +goose Down
ALTER TABLE jobs DROP COLUMN closed_at;
