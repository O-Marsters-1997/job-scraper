-- +goose Up
ALTER TABLE job_scores ADD COLUMN band TEXT CHECK (band IN ('great', 'good', 'fair', 'poor'));

-- +goose Down
ALTER TABLE job_scores DROP COLUMN band;
