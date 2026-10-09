-- +goose Up
ALTER TABLE source_targets
    DROP COLUMN check_interval_minutes,
    DROP COLUMN last_checked_at,
    ADD COLUMN interval_minutes INT CHECK (interval_minutes IS NULL OR interval_minutes >= 60),
    ADD COLUMN weekdays SMALLINT NOT NULL DEFAULT 31 CHECK (weekdays BETWEEN 1 AND 127),
    ADD COLUMN window_start TIME NOT NULL DEFAULT '08:00',
    ADD COLUMN window_end TIME NOT NULL DEFAULT '18:00',
    ADD COLUMN timezone TEXT NOT NULL DEFAULT 'Europe/London',
    ADD COLUMN next_run_at TIMESTAMPTZ,
    ADD CONSTRAINT source_targets_window_order CHECK (window_start < window_end);

CREATE INDEX source_targets_due_idx ON source_targets (next_run_at)
    WHERE enabled AND interval_minutes IS NOT NULL;

-- +goose Down
DROP INDEX source_targets_due_idx;

ALTER TABLE source_targets
    DROP CONSTRAINT source_targets_window_order,
    DROP COLUMN next_run_at,
    DROP COLUMN timezone,
    DROP COLUMN window_end,
    DROP COLUMN window_start,
    DROP COLUMN weekdays,
    DROP COLUMN interval_minutes,
    ADD COLUMN last_checked_at TIMESTAMPTZ,
    ADD COLUMN check_interval_minutes INT NOT NULL DEFAULT 360;
