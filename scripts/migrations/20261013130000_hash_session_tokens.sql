-- +goose Up
DELETE FROM sessions;
ALTER TABLE sessions ADD COLUMN token_hash BYTEA NOT NULL
    CONSTRAINT sessions_token_hash_key UNIQUE;

-- +goose Down
ALTER TABLE sessions DROP COLUMN token_hash;
