-- +goose Up
CREATE TABLE IF NOT EXISTS notification_digests (
    id        UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    sent_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    job_count INT         NOT NULL
);

-- +goose Down
DROP TABLE IF EXISTS notification_digests;
