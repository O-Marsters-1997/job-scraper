-- +goose Up
ALTER TABLE company_boards ADD COLUMN uk_jobs_seen_at TIMESTAMPTZ;

-- +goose Down
ALTER TABLE company_boards DROP COLUMN uk_jobs_seen_at;
