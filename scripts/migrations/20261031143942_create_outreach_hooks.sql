-- +goose Up
CREATE TABLE outreach_hooks (
    contact_id UUID        PRIMARY KEY REFERENCES contacts(id) ON DELETE CASCADE,
    text       TEXT        NOT NULL CHECK (length(text) BETWEEN 1 AND 600),
    model      TEXT        NOT NULL,
    cost       REAL        NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE outreach_hooks;
