-- +goose Up
CREATE TABLE contacts (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name         TEXT        NOT NULL CHECK (length(name) BETWEEN 1 AND 200),
    role         TEXT        NOT NULL DEFAULT '' CHECK (length(role) <= 200),
    company_id   UUID        REFERENCES companies(id) ON DELETE SET NULL,
    company_name TEXT        NOT NULL DEFAULT '' CHECK (length(company_name) <= 200),
    job_id       UUID        REFERENCES jobs(id) ON DELETE SET NULL,
    linkedin_url TEXT        CHECK (linkedin_url ~ '^https?://' AND length(linkedin_url) <= 2048),
    email        TEXT        CHECK (length(email) <= 320),
    stage        TEXT        NOT NULL DEFAULT 'to_contact'
        CONSTRAINT contacts_stage_check CHECK (stage IN ('to_contact', 'messaged', 'replied', 'closed')),
    outcome      TEXT
        CONSTRAINT contacts_outcome_check
        CHECK (outcome IN ('interested', 'referred', 'not_now', 'no_thanks', 'no_reply')),
    follow_up_by DATE,
    reply        TEXT        CHECK (length(reply) <= 4000),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX contacts_user_follow_up_idx ON contacts (user_id, follow_up_by) WHERE follow_up_by IS NOT NULL;
CREATE INDEX contacts_company_idx ON contacts (company_id) WHERE company_id IS NOT NULL;
CREATE INDEX contacts_job_idx ON contacts (job_id) WHERE job_id IS NOT NULL;

CREATE TABLE contact_events (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    contact_id    UUID        NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
    kind          TEXT        NOT NULL
        CONSTRAINT contact_events_kind_check CHECK (kind IN ('added', 'sent', 'reply', 'closed')),
    occurred_on   DATE        NOT NULL,
    channel       TEXT
        CONSTRAINT contact_events_channel_check CHECK (channel IN ('linkedin', 'email', 'call', 'inbound')),
    template_kind TEXT
        CONSTRAINT contact_events_template_kind_check
        CHECK (template_kind IN ('recruiter', 'hiring_manager', 'referral', 'follow_up')),
    template_name TEXT        NOT NULL DEFAULT '' CHECK (length(template_name) <= 100),
    outcome       TEXT
        CONSTRAINT contact_events_outcome_check
        CHECK (outcome IN ('interested', 'referred', 'not_now', 'no_thanks', 'no_reply')),
    note          TEXT        NOT NULL DEFAULT '' CHECK (length(note) <= 200),
    job_id        UUID        REFERENCES jobs(id) ON DELETE SET NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX contact_events_contact_idx ON contact_events (contact_id, occurred_on, created_at);
CREATE INDEX contact_events_job_idx ON contact_events (job_id) WHERE job_id IS NOT NULL;

-- +goose Down
DROP TABLE contact_events;
DROP TABLE contacts;
