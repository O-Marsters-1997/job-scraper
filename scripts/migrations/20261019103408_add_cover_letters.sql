-- +goose Up
CREATE TABLE cover_letters (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    job_id          UUID        NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    source_doc_id   TEXT        NOT NULL,
    source_tab_id   TEXT        NOT NULL,
    scaffold_tab_id TEXT        NOT NULL,
    achievement_ids UUID[]      NOT NULL,
    model           TEXT        NOT NULL,
    cost            REAL        NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX cover_letters_user_job_idx ON cover_letters (user_id, job_id, created_at DESC);

-- +goose Down
DROP TABLE cover_letters;
