-- +goose Up
UPDATE source_targets
SET filters = (filters - 'region') || '{"loc":"86383"}'
WHERE source = 'wis' AND filters->>'region' = 'uk';

-- +goose Down
UPDATE source_targets
SET filters = (filters - 'loc') || '{"region":"uk"}'
WHERE source = 'wis' AND filters->>'loc' = '86383';
