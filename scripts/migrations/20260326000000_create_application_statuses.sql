-- +goose Up
CREATE TABLE application_statuses (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name       TEXT        NOT NULL,
    colour     TEXT        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE application_statuses;
