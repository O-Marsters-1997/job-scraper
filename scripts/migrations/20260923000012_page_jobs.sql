-- +goose Up
CREATE INDEX jobs_page_idx ON jobs (scraped_at DESC, id DESC);
CREATE INDEX jobs_open_page_idx ON jobs (scraped_at DESC, id DESC) WHERE closed_at IS NULL;
CREATE INDEX jobs_company_page_idx ON jobs (company_id, scraped_at DESC, id DESC);
CREATE INDEX jobs_legacy_company_page_idx ON jobs (company_slug, scraped_at DESC, id DESC) WHERE company_id IS NULL;

-- +goose Down
DROP INDEX jobs_legacy_company_page_idx;
DROP INDEX jobs_company_page_idx;
DROP INDEX jobs_open_page_idx;
DROP INDEX jobs_page_idx;
