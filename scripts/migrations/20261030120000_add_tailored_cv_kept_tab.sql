-- +goose Up
ALTER TABLE tailored_cvs ADD COLUMN IF NOT EXISTS kept_tab_id TEXT;

-- +goose Down
ALTER TABLE tailored_cvs DROP COLUMN IF EXISTS kept_tab_id;
