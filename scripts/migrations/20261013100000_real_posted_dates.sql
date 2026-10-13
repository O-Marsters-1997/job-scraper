-- +goose Up
ALTER TABLE jobs RENAME COLUMN updated_at TO posted_at;
ALTER TABLE jobs ALTER COLUMN posted_at DROP NOT NULL;
ALTER TABLE jobs ADD COLUMN posted_after TIMESTAMPTZ;
ALTER TABLE jobs ADD COLUMN reposted_at TIMESTAMPTZ;

UPDATE jobs SET posted_at = NULL
WHERE abs(extract(epoch FROM posted_at - first_discovered_at)) < 120;

-- +goose Down
UPDATE jobs SET posted_at = first_discovered_at WHERE posted_at IS NULL;
ALTER TABLE jobs DROP COLUMN reposted_at;
ALTER TABLE jobs DROP COLUMN posted_after;
ALTER TABLE jobs ALTER COLUMN posted_at SET NOT NULL;
ALTER TABLE jobs RENAME COLUMN posted_at TO updated_at;
