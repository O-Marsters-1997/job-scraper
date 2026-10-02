-- +goose Up
ALTER TABLE source_targets ADD COLUMN last_succeeded_at TIMESTAMPTZ;
UPDATE source_targets SET last_succeeded_at = last_run_at WHERE run_status = 'succeeded';

-- +goose Down
ALTER TABLE source_targets DROP COLUMN last_succeeded_at;
