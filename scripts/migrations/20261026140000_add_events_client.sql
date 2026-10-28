-- +goose Up
ALTER TABLE events
    ADD COLUMN client TEXT NOT NULL DEFAULT 'web' CONSTRAINT events_client_check CHECK (client IN ('web', 'ios'));

-- +goose Down
ALTER TABLE events DROP COLUMN client;
