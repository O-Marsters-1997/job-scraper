-- +goose Up
ALTER TABLE job_scores
    ADD COLUMN blocked BOOLEAN NOT NULL GENERATED ALWAYS AS (breakdown @> '[{"effect":"blocked"}]'::jsonb) STORED;

-- +goose Down
ALTER TABLE job_scores DROP COLUMN blocked;
