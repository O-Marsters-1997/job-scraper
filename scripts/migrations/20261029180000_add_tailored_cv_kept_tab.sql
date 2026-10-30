-- +goose Up
ALTER TABLE tailored_cvs ADD COLUMN kept_tab_id TEXT;

-- +goose Down
ALTER TABLE tailored_cvs DROP COLUMN kept_tab_id;
