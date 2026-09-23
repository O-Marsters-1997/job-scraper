-- +goose Up
ALTER TABLE jobs
    ADD COLUMN company_id UUID REFERENCES companies(id) ON DELETE SET NULL,
    ADD COLUMN primary_board_id UUID,
    ADD COLUMN provider_posting_id TEXT,
    ADD COLUMN content_fingerprint TEXT,
    ADD COLUMN content_changed_at TIMESTAMPTZ,
    ADD COLUMN first_discovered_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

UPDATE jobs SET first_discovered_at = scraped_at;

CREATE UNIQUE INDEX jobs_board_posting_id_unique
    ON jobs(primary_board_id, provider_posting_id)
    WHERE primary_board_id IS NOT NULL AND provider_posting_id IS NOT NULL;

CREATE INDEX jobs_company_availability_idx ON jobs(company_id, closed_at);

CREATE TABLE job_urls (
    job_id UUID NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    normalized_url TEXT PRIMARY KEY,
    source TEXT NOT NULL,
    first_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO job_urls(job_id, normalized_url, source)
SELECT id, url, source FROM jobs;

CREATE INDEX job_urls_job_id_idx ON job_urls(job_id);

-- +goose Down
DROP TABLE job_urls;
DROP INDEX jobs_company_availability_idx;
DROP INDEX jobs_board_posting_id_unique;
ALTER TABLE jobs
    DROP COLUMN first_discovered_at,
    DROP COLUMN content_changed_at,
    DROP COLUMN content_fingerprint,
    DROP COLUMN provider_posting_id,
    DROP COLUMN primary_board_id,
    DROP COLUMN company_id;
