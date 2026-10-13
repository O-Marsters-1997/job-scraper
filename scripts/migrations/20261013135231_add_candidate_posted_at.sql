-- +goose Up
ALTER TABLE job_candidates ADD COLUMN posted_at TIMESTAMPTZ;

-- +goose Down
ALTER TABLE job_candidates DROP COLUMN posted_at;
