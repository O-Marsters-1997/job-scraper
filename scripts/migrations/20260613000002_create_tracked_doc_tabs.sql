-- +goose Up
CREATE TABLE tracked_doc_tabs (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tracked_doc_id UUID        NOT NULL REFERENCES tracked_docs(id) ON DELETE CASCADE,
    tab_id         TEXT        NOT NULL,
    title          TEXT        NOT NULL DEFAULT '',
    visible        BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tracked_doc_id, tab_id)
);
-- +goose Down
DROP TABLE tracked_doc_tabs;
