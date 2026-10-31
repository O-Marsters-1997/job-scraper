-- +goose Up
DROP TABLE IF EXISTS recruiter_contacts;

-- +goose Down
CREATE TABLE IF NOT EXISTS recruiter_contacts (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    job_id       UUID        NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    contacted_on DATE        NOT NULL,
    channel      TEXT        NOT NULL
        CONSTRAINT recruiter_contacts_channel_check
        CHECK (channel IN ('linkedin_message', 'email', 'call', 'inbound')),
    outcome      TEXT        NOT NULL
        CONSTRAINT recruiter_contacts_outcome_check
        CHECK (outcome IN ('no_reply', 'rejected', 'spoke', 'submitted')),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX recruiter_contacts_user_job_idx ON recruiter_contacts (user_id, job_id, contacted_on DESC);

