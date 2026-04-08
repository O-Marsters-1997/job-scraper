CREATE TABLE IF NOT EXISTS jobs (
    id           UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    title        TEXT         NOT NULL,
    location     TEXT         NOT NULL DEFAULT '',
    url          TEXT         NOT NULL UNIQUE,
    company_slug TEXT         NOT NULL,
    source       TEXT         NOT NULL,
    updated_at   TIMESTAMPTZ  NOT NULL,
    scraped_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS users (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    username      TEXT        NOT NULL UNIQUE,
    password_hash TEXT        NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS sessions (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS sessions_expires_at_idx ON sessions(expires_at);

CREATE TABLE IF NOT EXISTS application_statuses (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name       TEXT        NOT NULL,
    colour     TEXT        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS notification_digests (
    id        UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    sent_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    job_count INT         NOT NULL
);

CREATE TABLE IF NOT EXISTS applications (
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
