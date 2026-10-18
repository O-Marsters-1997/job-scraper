-- +goose Up
ALTER TABLE board_poll_state ADD COLUMN check_published_at TIMESTAMPTZ;

-- +goose Down
ALTER TABLE board_poll_state DROP COLUMN check_published_at;
