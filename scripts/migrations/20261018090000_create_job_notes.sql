-- +goose Up
CREATE TABLE job_notes (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    job_id     UUID        NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    body       TEXT        NOT NULL
        CONSTRAINT job_notes_body_check CHECK (length(body) BETWEEN 1 AND 10000),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX job_notes_user_job_idx ON job_notes (user_id, job_id, created_at DESC);

-- +goose Down
DROP TABLE job_notes;
