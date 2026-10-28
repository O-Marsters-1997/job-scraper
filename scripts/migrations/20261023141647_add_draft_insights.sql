-- +goose Up
ALTER TABLE tailored_cvs
    ADD COLUMN insights     JSONB,
    ADD COLUMN insights_for TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE tailored_cvs
    DROP COLUMN insights_for,
    DROP COLUMN insights;
