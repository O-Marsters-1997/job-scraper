-- +goose Up
ALTER TABLE tailored_cvs ADD COLUMN base_content JSONB;

-- +goose Down
ALTER TABLE tailored_cvs DROP COLUMN base_content;
