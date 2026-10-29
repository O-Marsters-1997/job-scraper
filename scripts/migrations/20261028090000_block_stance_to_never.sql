-- +goose Up
UPDATE search_config sc
SET preferences = jsonb_set(sc.preferences, '{picks}', (
    SELECT jsonb_agg(
        CASE WHEN p.pick->>'stance' = 'block' THEN p.pick || '{"stance": "never"}'::jsonb ELSE p.pick END
        ORDER BY p.ord)
    FROM jsonb_array_elements(sc.preferences->'picks') WITH ORDINALITY AS p(pick, ord)
))
WHERE jsonb_path_exists(sc.preferences, '$.picks[*] ? (@.stance == "block")');

-- +goose Down
SELECT 1;
