-- +goose Up
CREATE TABLE job_views (
    user_id UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    job_id  UUID        NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    seen_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, job_id)
);

-- +goose Down
DROP TABLE job_views;
