-- +goose Up
ALTER TABLE notifications ADD COLUMN kind TEXT NOT NULL DEFAULT 'new_match'
    CONSTRAINT notifications_kind_check CHECK (kind IN ('new_match'));

-- +goose Down
ALTER TABLE notifications DROP COLUMN kind;
