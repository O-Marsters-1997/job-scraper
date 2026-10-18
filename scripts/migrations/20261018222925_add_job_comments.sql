-- +goose Up
CREATE TABLE job_comments (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    job_id       UUID        NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    body         TEXT        NOT NULL
        CONSTRAINT job_comments_body_check CHECK (length(body) BETWEEN 1 AND 10000),
    quote        TEXT        NOT NULL
        CONSTRAINT job_comments_quote_check CHECK (length(quote) BETWEEN 1 AND 2000),
    quote_prefix TEXT        NOT NULL DEFAULT ''
        CONSTRAINT job_comments_quote_prefix_check CHECK (length(quote_prefix) <= 64),
    quote_suffix TEXT        NOT NULL DEFAULT ''
        CONSTRAINT job_comments_quote_suffix_check CHECK (length(quote_suffix) <= 64),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX job_comments_user_job_idx ON job_comments (user_id, job_id, created_at DESC);
CREATE INDEX job_comments_job_idx ON job_comments (job_id);

INSERT INTO job_comments (id, user_id, job_id, body, quote, quote_prefix, quote_suffix, created_at, updated_at)
SELECT id, user_id, job_id, body, quote, quote_prefix, quote_suffix, created_at, updated_at
FROM job_notes
WHERE quote IS NOT NULL;

DELETE FROM job_notes WHERE quote IS NOT NULL;

ALTER TABLE job_notes
    DROP COLUMN quote_suffix,
    DROP COLUMN quote_prefix,
    DROP COLUMN quote;

-- +goose Down
ALTER TABLE job_notes
    ADD COLUMN quote        TEXT
        CONSTRAINT job_notes_quote_check CHECK (length(quote) BETWEEN 1 AND 2000),
    ADD COLUMN quote_prefix TEXT NOT NULL DEFAULT ''
        CONSTRAINT job_notes_quote_prefix_check CHECK (length(quote_prefix) <= 64),
    ADD COLUMN quote_suffix TEXT NOT NULL DEFAULT ''
        CONSTRAINT job_notes_quote_suffix_check CHECK (length(quote_suffix) <= 64);

INSERT INTO job_notes (id, user_id, job_id, body, quote, quote_prefix, quote_suffix, created_at, updated_at)
SELECT id, user_id, job_id, body, quote, quote_prefix, quote_suffix, created_at, updated_at
FROM job_comments;

DROP TABLE job_comments;
