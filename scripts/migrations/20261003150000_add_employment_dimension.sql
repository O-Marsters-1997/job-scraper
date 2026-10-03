-- +goose NO TRANSACTION
-- +goose Up
ALTER TYPE scoring_dimension ADD VALUE IF NOT EXISTS 'employment';

-- +goose Down
SELECT 1;
