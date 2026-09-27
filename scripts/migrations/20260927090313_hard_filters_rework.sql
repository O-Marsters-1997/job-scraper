-- +goose Up

ALTER TABLE search_config DROP COLUMN excluded_title_keywords;

ALTER TABLE job_scores ADD COLUMN hidden BOOLEAN NOT NULL DEFAULT false;

-- +goose Down

ALTER TABLE job_scores DROP COLUMN hidden;

ALTER TABLE search_config ADD COLUMN excluded_title_keywords TEXT[] NOT NULL DEFAULT '{}';
