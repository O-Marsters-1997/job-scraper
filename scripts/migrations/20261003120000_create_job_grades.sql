-- +goose Up
CREATE TABLE job_grades (
    user_id        UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    job_id         UUID        NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    grade          TEXT        NOT NULL CHECK (grade IN ('great', 'ok', 'no')),
    reasons        TEXT[]      NOT NULL DEFAULT '{}',
    score_at_grade INT,
    score_model    TEXT,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, job_id)
);

-- +goose Down
DROP TABLE job_grades;
