-- +goose Up
CREATE TYPE event_type AS ENUM (
    'job_opened',
    'job_dismissed',
    'application_created',
    'application_status_changed',
    'alert_opened'
);

CREATE TABLE events (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type       event_type  NOT NULL,
    subject_id UUID,
    props      JSONB       NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX events_user_type_created_idx ON events (user_id, type, created_at);
CREATE INDEX events_subject_idx ON events (subject_id);

-- +goose Down
DROP TABLE events;
DROP TYPE event_type;
