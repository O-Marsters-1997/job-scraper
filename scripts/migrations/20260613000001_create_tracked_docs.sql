-- +goose Up
CREATE TABLE tracked_docs (
    id       UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id  UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    doc_id   TEXT        NOT NULL,
    added_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, doc_id)
);
-- +goose Down
DROP TABLE tracked_docs;
