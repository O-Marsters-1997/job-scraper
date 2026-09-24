-- +goose Up
ALTER TABLE source_targets ADD COLUMN run_id UUID;
CREATE TABLE harvest_runs (
    harvester TEXT PRIMARY KEY,
    last_succeeded_at TIMESTAMPTZ NOT NULL
);

-- +goose Down
DROP TABLE harvest_runs;
ALTER TABLE source_targets DROP COLUMN run_id;
