-- +goose Up
ALTER TABLE job_scores ADD COLUMN IF NOT EXISTS suitability_skipped BOOLEAN NOT NULL DEFAULT false;

-- +goose Down
ALTER TABLE job_scores DROP COLUMN IF EXISTS suitability_skipped;
