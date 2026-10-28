-- +goose Up
CREATE TABLE job_rejections (
    user_id               UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    job_id                UUID        NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    kind                  TEXT        NOT NULL
        CONSTRAINT job_rejections_kind_check CHECK (kind IN ('filter', 'age')),
    reason                TEXT        NOT NULL,
    search_config_version TIMESTAMPTZ,
    content_fingerprint   TEXT        NOT NULL,
    rejected_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, job_id)
);
CREATE INDEX job_rejections_job_idx ON job_rejections (job_id);
CREATE INDEX job_rejections_user_rejected_idx ON job_rejections (user_id, rejected_at DESC, job_id DESC);

-- +goose Down
DROP TABLE job_rejections;
