-- +goose Up
ALTER TABLE users ADD COLUMN cv_filename_pattern TEXT NOT NULL DEFAULT 'cv-{company}';

-- +goose Down
ALTER TABLE users DROP COLUMN cv_filename_pattern;
