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
    closed_at           TIMESTAMPTZ,
    company_id          UUID,
    primary_board_id    UUID,
    provider_posting_id TEXT,
    content_fingerprint TEXT,
    content_changed_at  TIMESTAMPTZ,
    first_discovered_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    match_title         TEXT,
    match_location      TEXT
);

CREATE INDEX jobs_page_idx ON jobs (scraped_at DESC, id DESC);
CREATE INDEX jobs_open_page_idx ON jobs (scraped_at DESC, id DESC) WHERE closed_at IS NULL;
CREATE INDEX jobs_company_page_idx ON jobs (company_id, scraped_at DESC, id DESC);
CREATE INDEX jobs_legacy_company_page_idx ON jobs (company_slug, scraped_at DESC, id DESC) WHERE company_id IS NULL;
CREATE INDEX jobs_match_idx ON jobs (match_title, company_slug) WHERE closed_at IS NULL;

CREATE TABLE job_urls (
    job_id UUID NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    normalized_url TEXT PRIMARY KEY,
    source TEXT NOT NULL,
    first_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
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

CREATE TABLE IF NOT EXISTS google_oauth_tokens (
    id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id           UUID        NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    access_token_enc  TEXT        NOT NULL,
    refresh_token_enc TEXT        NOT NULL,
    token_type        TEXT        NOT NULL,
    expiry            TIMESTAMPTZ,
    scope             TEXT        NOT NULL,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS application_statuses (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name       TEXT        NOT NULL,
    colour     TEXT        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    reply_window_days INT  CHECK (reply_window_days BETWEEN 1 AND 60)
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
    chase_by    DATE,
    UNIQUE (user_id, job_id)
);

CREATE INDEX applications_chase_idx ON applications (user_id, chase_by) WHERE chase_by IS NOT NULL;

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

CREATE TABLE positions (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    employer   TEXT        NOT NULL,
    title      TEXT        NOT NULL,
    start_date DATE,
    end_date   DATE,
    sort_order INT         NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX positions_user_sort_idx ON positions (user_id, sort_order);

CREATE TABLE achievements (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    position_id UUID        NOT NULL REFERENCES positions(id) ON DELETE CASCADE,
    user_id     UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    text        TEXT        NOT NULL,
    sort_order  INT         NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX achievements_position_sort_idx ON achievements (position_id, sort_order);

CREATE TABLE bank_skills (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name       TEXT        NOT NULL,
    category   TEXT        NOT NULL DEFAULT '',
    sort_order INT         NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX bank_skills_user_name_idx ON bank_skills (user_id, lower(name));
CREATE INDEX bank_skills_user_sort_idx ON bank_skills (user_id, sort_order);

CREATE TABLE cv_heading_mappings (
    user_id      UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    doc_id       TEXT        NOT NULL,
    tab_id       TEXT        NOT NULL,
    heading_text TEXT        NOT NULL,
    position_id  UUID        REFERENCES positions(id) ON DELETE SET NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, doc_id, tab_id, heading_text)
);

CREATE TABLE tailored_cvs (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    job_id          UUID        NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    base_doc_id     TEXT        NOT NULL,
    base_tab_id     TEXT        NOT NULL,
    achievement_ids UUID[]      NOT NULL,
    edit_set        JSONB,
    base_content    JSONB,
    findings        JSONB       NOT NULL DEFAULT '[]',
    raw_output      TEXT        NOT NULL DEFAULT '',
    model           TEXT        NOT NULL DEFAULT '',
    prompt_version  TEXT        NOT NULL DEFAULT '',
    job_fingerprint TEXT        NOT NULL DEFAULT '',
    cost            REAL        NOT NULL DEFAULT 0,
    draft_doc_id    TEXT,
    status          TEXT        NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'running', 'keeping', 'ready', 'failed')),
    outcome         TEXT        CHECK (outcome IN ('kept', 'discarded')),
    kept_as         TEXT        CHECK (kept_as IN ('tab', 'doc')),
    keep_note       TEXT        NOT NULL DEFAULT '',
    attempts        INT         NOT NULL DEFAULT 0,
    due_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    lease_until     TIMESTAMPTZ,
    last_error      TEXT        NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX tailored_cvs_kept_idx ON tailored_cvs (user_id, job_id) WHERE outcome = 'kept';
CREATE INDEX tailored_cvs_claim_idx ON tailored_cvs (due_at, id) WHERE status IN ('pending', 'running', 'keeping');

CREATE TABLE preference_labels (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id        UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    job_id         UUID        NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    draft_id       UUID        REFERENCES tailored_cvs(id) ON DELETE SET NULL,
    kind           TEXT        NOT NULL CHECK (kind IN ('bullet')),
    achievement_id UUID        REFERENCES achievements(id) ON DELETE SET NULL,
    p_yes          DOUBLE PRECISION,
    p_no           DOUBLE PRECISION,
    p_not_stated   DOUBLE PRECISION,
    confidence     DOUBLE PRECISION,
    preselected    BOOLEAN     NOT NULL,
    kept           BOOLEAN     NOT NULL,
    position       INT,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX preference_labels_user_job_idx ON preference_labels (user_id, job_id);
CREATE INDEX preference_labels_draft_idx ON preference_labels (draft_id);

CREATE TABLE IF NOT EXISTS job_scores (
    id                   UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id               UUID        NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    user_id              UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    suitability_score    INT,
    band                 TEXT        CHECK (band IN ('great', 'good', 'fair', 'poor')),
    breakdown            JSONB       NOT NULL DEFAULT '[]',
    cost                 NUMERIC,
    score_fingerprint    TEXT,
    score_model          TEXT,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (job_id, user_id)
);

CREATE TABLE effect_outbox (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id UUID NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    fingerprint TEXT NOT NULL,
    model TEXT NOT NULL DEFAULT 'typesafe/jev-1.13',
    first_discovery BOOLEAN NOT NULL DEFAULT FALSE,
    status TEXT NOT NULL DEFAULT 'pending',
    attempts INT NOT NULL DEFAULT 0,
    due_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    lease_until TIMESTAMPTZ,
    last_error TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX effect_outbox_pending_idx ON effect_outbox (job_id, fingerprint, model)
    WHERE status IN ('pending', 'running');
CREATE INDEX job_scores_user_job_idx ON job_scores (user_id, job_id);

CREATE TABLE answer_corrections (
    user_id    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    job_id     UUID        NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    option_id  TEXT        NOT NULL REFERENCES scoring_options(id),
    value      TEXT        NOT NULL CHECK (value IN ('yes', 'no')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, job_id, option_id)
);

CREATE TABLE IF NOT EXISTS search_config (
    id                      UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id                 UUID        NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    excluded_title_keywords TEXT[]      NOT NULL DEFAULT '{}',
    excluded_companies      TEXT[]      NOT NULL DEFAULT '{}',
    excluded_locations      TEXT[]      NOT NULL DEFAULT '{}',
    required_locations      TEXT[]      NOT NULL DEFAULT '{}',
    required_title_keywords TEXT[]      NOT NULL DEFAULT '{}',
    notify_threshold        INT         NOT NULL DEFAULT 70,
    preferences             JSONB       NOT NULL DEFAULT '{}',
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
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
    run_id                 UUID,
    last_run_at            TIMESTAMPTZ,
    last_succeeded_at      TIMESTAMPTZ,
    last_run_error         TEXT        NOT NULL DEFAULT '',
    disabled_reason        TEXT        NOT NULL DEFAULT '',
    created_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, source, value, filters)
);

CREATE TABLE IF NOT EXISTS tracked_companies (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    check_interval_minutes INT NOT NULL DEFAULT 360 CHECK (check_interval_minutes >= 60),
    review_state TEXT NOT NULL DEFAULT 'kept' CHECK (review_state IN ('new', 'kept', 'dismissed')),
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
    superseded_at TIMESTAMPTZ,
    discovered_via TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (source, board_token)
);

CREATE TABLE board_poll_state (
    board_id UUID PRIMARY KEY REFERENCES company_boards(id) ON DELETE CASCADE,
    last_completed_at TIMESTAMPTZ,
    last_scheduled_at TIMESTAMPTZ,
    last_started_at TIMESTAMPTZ,
    last_snapshot_version BIGINT NOT NULL DEFAULT 0,
    consecutive_complete_empty INT NOT NULL DEFAULT 0,
    consecutive_failures INT NOT NULL DEFAULT 0,
    last_reported_total INT NOT NULL DEFAULT 0,
    last_parsed INT NOT NULL DEFAULT 0,
    lease_owner TEXT,
    lease_until TIMESTAMPTZ,
    next_due_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE board_job_observations (
    board_id UUID NOT NULL REFERENCES company_boards(id) ON DELETE CASCADE,
    job_id UUID NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_snapshot_version BIGINT NOT NULL,
    PRIMARY KEY (board_id, job_id)
);

CREATE INDEX board_poll_state_due_idx ON board_poll_state(next_due_at) WHERE lease_until IS NULL;
CREATE INDEX board_job_observations_version_idx ON board_job_observations(board_id, last_snapshot_version);

CREATE TABLE IF NOT EXISTS job_candidates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    normalized_url TEXT NOT NULL UNIQUE,
    source TEXT NOT NULL,
    card_title TEXT NOT NULL DEFAULT '',
    card_company TEXT NOT NULL DEFAULT '',
    card_location TEXT NOT NULL DEFAULT '',
    card JSONB,
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

CREATE TABLE harvest_runs (
    harvester TEXT PRIMARY KEY,
    last_succeeded_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS candidate_assessments (
    candidate_id UUID NOT NULL REFERENCES job_candidates(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    search_config_version TIMESTAMPTZ NOT NULL,
    relevance BOOLEAN NOT NULL,
    evaluated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (candidate_id, user_id)
);

CREATE TYPE scoring_dimension AS ENUM ('tech', 'role', 'domain', 'seniority', 'work', 'stage', 'size', 'employment');

CREATE TABLE scoring_options (
    id         TEXT PRIMARY KEY,
    dimension  scoring_dimension NOT NULL,
    label      TEXT NOT NULL,
    question   TEXT NOT NULL,
    retired_at TIMESTAMPTZ
);

CREATE TABLE option_answers (
    job_id        UUID NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    fingerprint   TEXT NOT NULL,
    question_hash TEXT NOT NULL,
    model         TEXT NOT NULL,
    p_yes         REAL NOT NULL,
    p_no          REAL NOT NULL,
    p_not_stated  REAL NOT NULL,
    confidence    REAL NOT NULL,
    answered_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (job_id, fingerprint, question_hash, model)
);

CREATE TABLE fetch_cache (
    url        TEXT PRIMARY KEY,
    status     INT NOT NULL,
    header     JSONB NOT NULL,
    body       BYTEA NOT NULL,
    fetched_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX fetch_cache_fetched_at_idx ON fetch_cache (fetched_at);

CREATE TABLE company_profiles (
    company_id UUID        NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    source     TEXT        NOT NULL,
    data       JSONB       NOT NULL,
    fetched_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (company_id, source)
);

CREATE TABLE push_subscriptions (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    endpoint   TEXT        NOT NULL UNIQUE,
    p256dh     TEXT        NOT NULL,
    auth       TEXT        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX push_subscriptions_user_id_idx ON push_subscriptions (user_id);

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

CREATE TABLE job_grades (
    user_id        UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    job_id         UUID        NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    grade          TEXT        NOT NULL CHECK (grade IN ('great', 'ok', 'no')),
    reasons        TEXT[]      NOT NULL DEFAULT '{}',
    score_at_grade INT,
    score_model    TEXT,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, job_id)
);

CREATE TABLE job_views (
    user_id UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    job_id  UUID        NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    seen_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, job_id)
);

CREATE TABLE company_favourites (
    user_id    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    company_id UUID        NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, company_id)
);
