-- +goose Up
ALTER TABLE users ADD COLUMN IF NOT EXISTS cv_filename_pattern TEXT NOT NULL DEFAULT 'cv-{company}';

-- +goose Down
ALTER TABLE users DROP COLUMN IF EXISTS cv_filename_pattern;
