-- +goose Up
ALTER TABLE application_statuses
    ADD COLUMN reply_window_days INT CHECK (reply_window_days BETWEEN 1 AND 60);

-- +goose Down
ALTER TABLE application_statuses DROP COLUMN reply_window_days;
