-- +goose Up
CREATE TABLE wttj_companies (
    url_safe_name TEXT PRIMARY KEY,
    company_id    UUID REFERENCES companies (id) ON DELETE SET NULL,
    uk            BOOLEAN NOT NULL DEFAULT FALSE,
    live_jobs     INT NOT NULL DEFAULT 0,
    fetched_at    TIMESTAMPTZ,
    next_fetch_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    first_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX wttj_companies_next_fetch_at_idx ON wttj_companies (next_fetch_at);

CREATE TABLE wttj_jobs (
    job_id        TEXT PRIMARY KEY,
    wttj_company  TEXT REFERENCES wttj_companies (url_safe_name),
    first_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_seen_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    gone_at       TIMESTAMPTZ,
    state         TEXT NOT NULL DEFAULT 'seen' CHECK (state IN ('seen', 'covered', 'resolved', 'ingested', 'skipped'))
);

-- +goose Down
DROP TABLE wttj_jobs;
DROP TABLE wttj_companies;
