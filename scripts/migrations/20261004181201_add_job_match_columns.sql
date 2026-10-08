-- +goose Up
ALTER TABLE jobs ADD COLUMN match_title TEXT, ADD COLUMN match_location TEXT;
CREATE INDEX jobs_match_idx ON jobs (match_title, company_slug) WHERE closed_at IS NULL;

-- +goose Down
DROP INDEX jobs_match_idx;
ALTER TABLE jobs DROP COLUMN match_title, DROP COLUMN match_location;
