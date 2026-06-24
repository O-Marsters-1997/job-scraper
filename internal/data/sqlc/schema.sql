CREATE TABLE IF NOT EXISTS jobs (
    id           UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    title        TEXT         NOT NULL,
    location     TEXT         NOT NULL DEFAULT '',
    url          TEXT         NOT NULL UNIQUE,
    company_slug TEXT         NOT NULL,
    source       TEXT         NOT NULL,
    updated_at   TIMESTAMPTZ  NOT NULL,
    scraped_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    description      TEXT         NOT NULL DEFAULT '',
    salary_raw       TEXT         NOT NULL DEFAULT '',
    work_arrangement TEXT         NOT NULL DEFAULT ''
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

CREATE TABLE IF NOT EXISTS tracked_docs (
    id       UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id  UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    doc_id   TEXT        NOT NULL,
    added_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, doc_id)
);

CREATE TABLE IF NOT EXISTS tracked_doc_tabs (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tracked_doc_id UUID        NOT NULL REFERENCES tracked_docs(id) ON DELETE CASCADE,
    tab_id         TEXT        NOT NULL,
    title          TEXT        NOT NULL DEFAULT '',
    visible        BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tracked_doc_id, tab_id)
);

CREATE TABLE IF NOT EXISTS job_scores (
    id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id            UUID        NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    user_id           UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    relevance_score   INT,
    suitability_score INT,
    reasoning         TEXT,
    matched           TEXT[],
    missing           TEXT[],
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (job_id, user_id)
);

CREATE TABLE IF NOT EXISTS search_config (
    id                 UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id            UUID        NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    role               TEXT        NOT NULL DEFAULT '',
    location           TEXT        NOT NULL DEFAULT '',
    keywords           TEXT[]      NOT NULL DEFAULT '{}',
    suitability_rubric TEXT        NOT NULL DEFAULT '',
    relevance_cutoff   INT         NOT NULL DEFAULT 0,
    notify_threshold   INT         NOT NULL DEFAULT 70,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS user_ai_prefs (
    id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id           UUID        UNIQUE NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    suitability_model TEXT        NOT NULL DEFAULT 'claude-haiku-4-5-20251001',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS source_targets (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    source     TEXT        NOT NULL,
    value      TEXT        NOT NULL,
    enabled    BOOLEAN     NOT NULL DEFAULT TRUE,
    filters    JSONB       NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, source, value, filters)
);
