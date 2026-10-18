-- +goose Up
CREATE TABLE board_discover_misses (
    source      TEXT        NOT NULL,
    board_token TEXT        NOT NULL,
    missed_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (source, board_token)
);

-- +goose Down
DROP TABLE board_discover_misses;
