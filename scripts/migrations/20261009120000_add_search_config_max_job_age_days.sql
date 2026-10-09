-- +goose Up
ALTER TABLE search_config ADD COLUMN max_job_age_days INT NOT NULL DEFAULT 7
    CHECK (max_job_age_days BETWEEN 0 AND 365);

-- +goose Down
ALTER TABLE search_config DROP COLUMN max_job_age_days;
