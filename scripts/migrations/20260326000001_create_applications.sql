-- +goose Up
CREATE TABLE applications (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    job_id      UUID        NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    status_id   UUID        REFERENCES application_statuses(id) ON DELETE SET NULL,
    notes       TEXT,
    applied_at  DATE,
    salary_info TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, job_id)
);

-- +goose Down
DROP TABLE applications;
