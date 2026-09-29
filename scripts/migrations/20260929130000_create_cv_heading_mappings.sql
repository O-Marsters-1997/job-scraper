-- +goose Up
CREATE TABLE cv_heading_mappings (
    user_id      UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    doc_id       TEXT        NOT NULL,
    tab_id       TEXT        NOT NULL,
    heading_text TEXT        NOT NULL,
    position_id  UUID        REFERENCES positions(id) ON DELETE SET NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, doc_id, tab_id, heading_text)
);

-- +goose Down
DROP TABLE cv_heading_mappings;
