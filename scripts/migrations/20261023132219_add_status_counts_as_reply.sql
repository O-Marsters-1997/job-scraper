-- +goose Up
ALTER TABLE application_statuses
    ADD COLUMN counts_as_reply BOOLEAN NOT NULL DEFAULT FALSE;

-- +goose Down
ALTER TABLE application_statuses DROP COLUMN counts_as_reply;
