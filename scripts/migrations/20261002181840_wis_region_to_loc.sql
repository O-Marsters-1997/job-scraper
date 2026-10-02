-- +goose Up
UPDATE source_targets t
SET filters = (t.filters - 'region') || '{"loc":"86383"}'
WHERE t.source = 'wis' AND t.filters->>'region' = 'uk'
  AND NOT EXISTS (
    SELECT 1 FROM source_targets o
    WHERE o.user_id = t.user_id AND o.source = t.source AND o.value = t.value
      AND o.filters = (t.filters - 'region') || '{"loc":"86383"}'
  );

-- +goose Down
UPDATE source_targets t
SET filters = (t.filters - 'loc') || '{"region":"uk"}'
WHERE t.source = 'wis' AND t.filters->>'loc' = '86383'
  AND NOT EXISTS (
    SELECT 1 FROM source_targets o
    WHERE o.user_id = t.user_id AND o.source = t.source AND o.value = t.value
      AND o.filters = (t.filters - 'loc') || '{"region":"uk"}'
  );
