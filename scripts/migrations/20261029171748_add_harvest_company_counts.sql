-- +goose Up
CREATE TABLE harvest_company_counts (
    source     TEXT    NOT NULL,
    company    TEXT    NOT NULL,
    live_jobs  INTEGER NOT NULL CONSTRAINT harvest_company_counts_live_jobs_check CHECK (live_jobs >= 0),
    PRIMARY KEY (source, company)
);

ALTER TABLE harvest_seen_ids ADD COLUMN last_in_sitemap_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

-- +goose Down
ALTER TABLE harvest_seen_ids DROP COLUMN last_in_sitemap_at;
DROP TABLE harvest_company_counts;
