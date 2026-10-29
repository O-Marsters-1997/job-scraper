-- +goose NO TRANSACTION
-- +goose Up
ALTER TYPE event_type ADD VALUE IF NOT EXISTS 'job_impression';

-- +goose Down
SELECT 1;
