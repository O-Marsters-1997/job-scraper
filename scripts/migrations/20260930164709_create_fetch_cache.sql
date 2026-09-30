-- +goose Up
CREATE TABLE fetch_cache (
    url        TEXT PRIMARY KEY,
    status     INT NOT NULL,
    header     JSONB NOT NULL,
    body       BYTEA NOT NULL,
    fetched_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX fetch_cache_fetched_at_idx ON fetch_cache (fetched_at);

-- +goose Down
DROP TABLE fetch_cache;
