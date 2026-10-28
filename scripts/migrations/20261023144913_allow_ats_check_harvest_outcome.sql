-- +goose Up
ALTER TABLE harvest_seen_ids DROP CONSTRAINT harvest_seen_ids_outcome_check;
ALTER TABLE harvest_seen_ids ADD CONSTRAINT harvest_seen_ids_outcome_check CHECK (outcome IN ('backfill', 'ineligible', 'exported', 'ats_check'));

-- +goose Down
DELETE FROM harvest_seen_ids WHERE outcome = 'ats_check';
ALTER TABLE harvest_seen_ids DROP CONSTRAINT harvest_seen_ids_outcome_check;
ALTER TABLE harvest_seen_ids ADD CONSTRAINT harvest_seen_ids_outcome_check CHECK (outcome IN ('backfill', 'ineligible', 'exported'));
