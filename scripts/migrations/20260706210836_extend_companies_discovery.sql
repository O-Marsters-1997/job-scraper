-- +goose Up
ALTER TABLE companies
  ADD COLUMN domain TEXT,
  ADD COLUMN linkedin_company_id TEXT,
  ADD COLUMN last_crawled_at TIMESTAMPTZ;

-- +goose Down
ALTER TABLE companies
  DROP COLUMN domain,
  DROP COLUMN linkedin_company_id,
  DROP COLUMN last_crawled_at;
