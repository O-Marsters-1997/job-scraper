-- +goose Up
ALTER TABLE cover_letters ADD COLUMN done_at TIMESTAMPTZ;

-- +goose Down
ALTER TABLE cover_letters DROP COLUMN done_at;
