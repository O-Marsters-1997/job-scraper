-- +goose Up
CREATE TABLE letter_skeletons (
    user_id    UUID        PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    intro      TEXT        NOT NULL DEFAULT '',
    conclusion TEXT        NOT NULL DEFAULT '',
    sections   TEXT[]      NOT NULL DEFAULT '{}',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE letter_skeletons;
