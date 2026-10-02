-- +goose Up
CREATE TABLE score_feedback (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    job_id     UUID        REFERENCES jobs(id) ON DELETE SET NULL,
    kind       TEXT        NOT NULL CHECK (kind IN ('job', 'collection', 'overall')),
    direction  TEXT        CHECK (direction IN ('higher', 'lower')),
    reason     TEXT        NOT NULL CHECK (btrim(reason) <> ''),
    picks      JSONB       NOT NULL,
    model      TEXT        NOT NULL,
    snapshot   JSONB       NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK ((kind = 'job') = (direction IS NOT NULL))
);
CREATE INDEX score_feedback_user_created_idx ON score_feedback (user_id, created_at DESC);

-- +goose Down
DROP TABLE score_feedback;
