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
    work_arrangement TEXT         NOT NULL DEFAULT '',
    closed_at        TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS users (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    username      TEXT        NOT NULL UNIQUE,
    password_hash TEXT        NOT NULL,
    email         TEXT        UNIQUE,
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
    id                   UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id               UUID        NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    user_id              UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    relevance_score      INT,
    suitability_score    INT,
    reasoning            TEXT,
    matched              TEXT[],
    missing              TEXT[],
    suitability_skipped  BOOLEAN     NOT NULL DEFAULT false,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (job_id, user_id)
);

CREATE TABLE IF NOT EXISTS search_config (
    id                      UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id                 UUID        NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    excluded_title_keywords TEXT[]      NOT NULL DEFAULT '{}',
    excluded_companies      TEXT[]      NOT NULL DEFAULT '{}',
    excluded_seniority      TEXT[]      NOT NULL DEFAULT '{}',
    excluded_locations      TEXT[]      NOT NULL DEFAULT '{}',
    suitability_rubric      TEXT        NOT NULL DEFAULT '',
    notify_threshold        INT         NOT NULL DEFAULT 70,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS user_ai_prefs (
    id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id           UUID        UNIQUE NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    suitability_model TEXT        NOT NULL DEFAULT 'claude-haiku-4-5-20251001',
    reasoning_model   TEXT        NOT NULL DEFAULT 'claude-sonnet-4-6',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS user_ai_credentials (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider    TEXT        NOT NULL,
    api_key_enc TEXT        NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, provider)
);

CREATE TABLE IF NOT EXISTS companies (
    id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    slug                TEXT        NOT NULL UNIQUE,
    name                TEXT        NOT NULL,
    ats_source          TEXT,
    ats_token           TEXT,
    domain              TEXT,
    linkedin_company_id TEXT,
    last_crawled_at     TIMESTAMPTZ,
    first_seen_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS source_targets (
    id                     UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id                UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    source                 TEXT        NOT NULL,
    value                  TEXT        NOT NULL,
    enabled                BOOLEAN     NOT NULL DEFAULT TRUE,
    filters                JSONB       NOT NULL DEFAULT '{}',
    company_id             UUID        REFERENCES companies(id) ON DELETE SET NULL,
    check_interval_minutes INT         NOT NULL DEFAULT 360,
    last_checked_at        TIMESTAMPTZ,
    run_status             TEXT        NOT NULL DEFAULT 'idle' CHECK (run_status IN ('idle', 'queued', 'running', 'succeeded', 'failed')),
    last_run_at            TIMESTAMPTZ,
    last_run_error         TEXT        NOT NULL DEFAULT '',
    created_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, source, value, filters)
);

CREATE TABLE IF NOT EXISTS tracked_companies (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    check_interval_minutes INT NOT NULL DEFAULT 360 CHECK (check_interval_minutes >= 60),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, company_id)
);

CREATE TABLE IF NOT EXISTS company_boards (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    source TEXT NOT NULL,
    board_token TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'candidate' CHECK (status IN ('candidate', 'verified', 'retired')),
    verification_method TEXT,
    verified_at TIMESTAMPTZ,
    last_linked_at TIMESTAMPTZ,
    retired_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (source, board_token)
);

CREATE TABLE IF NOT EXISTS job_candidates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    normalized_url TEXT NOT NULL UNIQUE,
    source TEXT NOT NULL,
    card_title TEXT NOT NULL DEFAULT '',
    card_company TEXT NOT NULL DEFAULT '',
    card_location TEXT NOT NULL DEFAULT '',
    first_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL DEFAULT NOW() + INTERVAL '60 days',
    detail_state TEXT NOT NULL DEFAULT 'unrequested' CHECK (detail_state IN ('unrequested', 'pending'))
);

CREATE INDEX IF NOT EXISTS job_candidates_expires_at_idx ON job_candidates(expires_at);

CREATE TABLE IF NOT EXISTS candidate_discoveries (
    candidate_id UUID NOT NULL REFERENCES job_candidates(id) ON DELETE CASCADE,
    source_target_id UUID NOT NULL REFERENCES source_targets(id) ON DELETE CASCADE,
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (candidate_id, source_target_id)
);

CREATE INDEX IF NOT EXISTS candidate_discoveries_target_idx ON candidate_discoveries(source_target_id, candidate_id);

CREATE TABLE IF NOT EXISTS candidate_assessments (
    candidate_id UUID NOT NULL REFERENCES job_candidates(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    search_config_version TIMESTAMPTZ NOT NULL,
    relevance BOOLEAN NOT NULL,
    evaluated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (candidate_id, user_id)
);
