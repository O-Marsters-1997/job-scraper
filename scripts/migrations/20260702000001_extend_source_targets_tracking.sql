-- +goose Up
ALTER TABLE source_targets
    ADD COLUMN company_id             UUID REFERENCES companies(id) ON DELETE SET NULL,
    ADD COLUMN check_interval_minutes INT NOT NULL DEFAULT 360,
    ADD COLUMN last_checked_at        TIMESTAMPTZ;

-- +goose Down
ALTER TABLE source_targets
    DROP COLUMN company_id,
    DROP COLUMN check_interval_minutes,
    DROP COLUMN last_checked_at;
