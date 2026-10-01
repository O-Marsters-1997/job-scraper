-- +goose Up
ALTER TABLE job_candidates ADD COLUMN card JSONB;

-- +goose Down
ALTER TABLE job_candidates DROP COLUMN card;
