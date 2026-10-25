-- +goose Up
ALTER TABLE tracked_doc_tabs ADD COLUMN is_default BOOLEAN NOT NULL DEFAULT FALSE;

-- +goose Down
ALTER TABLE tracked_doc_tabs DROP COLUMN is_default;
