-- +goose Up
ALTER TABLE board_poll_state
    ADD COLUMN last_reported_total INT NOT NULL DEFAULT 0,
    ADD COLUMN last_parsed INT NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE board_poll_state
    DROP COLUMN last_parsed,
    DROP COLUMN last_reported_total;
