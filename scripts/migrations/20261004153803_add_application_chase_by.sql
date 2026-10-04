-- +goose Up
ALTER TABLE applications ADD COLUMN chase_by DATE;
CREATE INDEX applications_chase_idx ON applications (user_id, chase_by) WHERE chase_by IS NOT NULL;

-- +goose Down
DROP INDEX applications_chase_idx;
ALTER TABLE applications DROP COLUMN chase_by;
