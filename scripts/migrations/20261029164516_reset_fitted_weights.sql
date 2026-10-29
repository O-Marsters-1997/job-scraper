-- +goose Up
UPDATE search_config
SET preferences = preferences - 'scoring'
WHERE preferences ? 'scoring';

-- +goose Down
SELECT 1;
