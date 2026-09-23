-- +goose Up
ALTER TABLE source_targets
    ADD COLUMN run_status TEXT NOT NULL DEFAULT 'idle',
    ADD COLUMN last_run_at TIMESTAMPTZ,
    ADD COLUMN last_run_error TEXT NOT NULL DEFAULT '';

ALTER TABLE source_targets
    ADD CONSTRAINT source_targets_run_status_check
    CHECK (run_status IN ('idle', 'queued', 'running', 'succeeded', 'failed'));

-- +goose Down
ALTER TABLE source_targets DROP CONSTRAINT source_targets_run_status_check;
ALTER TABLE source_targets
    DROP COLUMN run_status,
    DROP COLUMN last_run_at,
    DROP COLUMN last_run_error;
