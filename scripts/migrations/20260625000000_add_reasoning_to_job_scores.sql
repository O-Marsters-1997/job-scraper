-- +goose Up
ALTER TABLE job_scores ADD COLUMN IF NOT EXISTS reasoning TEXT;
ALTER TABLE job_scores ADD COLUMN IF NOT EXISTS matched  TEXT[];
ALTER TABLE job_scores ADD COLUMN IF NOT EXISTS missing  TEXT[];

-- +goose Down
ALTER TABLE job_scores DROP COLUMN IF EXISTS reasoning;
ALTER TABLE job_scores DROP COLUMN IF EXISTS matched;
ALTER TABLE job_scores DROP COLUMN IF EXISTS missing;
