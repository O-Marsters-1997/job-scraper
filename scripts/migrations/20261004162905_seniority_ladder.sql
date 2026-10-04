-- +goose Up
UPDATE scoring_options SET question = 'Weighing title, required years and scope together, is the role Junior level (Junior or Graduate, about 0-2 years, works under guidance)?' WHERE id = 'seniority:junior';
UPDATE scoring_options SET question = 'Weighing title, required years and scope together, is the role Mid level (about 2-5 years, delivers independently, no Senior or higher title)?' WHERE id = 'seniority:mid';
UPDATE scoring_options SET question = 'Weighing title, required years and scope together, is the role Senior level (Senior title or about 5-8 years, owns features and mentors, not leading a team)?' WHERE id = 'seniority:senior';
UPDATE scoring_options SET retired_at = NOW() WHERE id = 'seniority:staff';
INSERT INTO scoring_options (id, dimension, label, question) VALUES
    ('seniority:lead_staff', 'seniority', 'Lead / Staff', 'Weighing title, required years and scope together, is the role Lead or Staff level (Lead, Staff or Tech Lead, about 8+ years, leads a team or owns cross-team technical scope)?'),
    ('seniority:principal_head', 'seniority', 'Principal / Head', 'Weighing title, required years and scope together, is the role Principal or Head level (Principal, Head of or Director, organisation-wide scope or manages managers)?')
ON CONFLICT (id) DO NOTHING;

UPDATE search_config sc
SET preferences = jsonb_set(sc.preferences, '{picks}', COALESCE((
    SELECT jsonb_agg(
        CASE
            WHEN p.pick->>'optionId' LIKE 'seniority:%' THEN p.pick || jsonb_build_object(
                'optionId', tier,
                'stance', 'nice',
                'source', 'manual',
                'weight', CASE p.pick->>'stance' WHEN 'nice' THEN 100 ELSE 50 END)
            ELSE p.pick
        END ORDER BY p.ord, tier)
    FROM jsonb_array_elements(sc.preferences->'picks') WITH ORDINALITY AS p(pick, ord)
    CROSS JOIN LATERAL unnest(CASE p.pick->>'optionId'
        WHEN 'seniority:staff' THEN ARRAY['seniority:lead_staff', 'seniority:principal_head']
        ELSE ARRAY[p.pick->>'optionId'] END) AS tier
    WHERE p.pick->>'optionId' NOT LIKE 'seniority:%'
        OR (p.pick->>'stance' IN ('nice', 'ok') AND NOT (p.pick->>'source' = 'text' AND sc.preferences->'picks' @> jsonb_build_array(
            jsonb_build_object('optionId', p.pick->>'optionId', 'source', 'manual'))))
), '[]'::jsonb))
WHERE jsonb_path_exists(sc.preferences, '$.picks[*] ? (@.optionId starts with "seniority:")');

-- +goose Down
UPDATE search_config sc
SET preferences = jsonb_set(sc.preferences, '{picks}', COALESCE((
    SELECT jsonb_agg(DISTINCT
        CASE
            WHEN pick->>'optionId' LIKE 'seniority:%' THEN (pick - 'weight') || jsonb_build_object(
                'optionId', CASE WHEN pick->>'optionId' IN ('seniority:lead_staff', 'seniority:principal_head')
                    THEN 'seniority:staff' ELSE pick->>'optionId' END,
                'stance', CASE WHEN (pick->>'weight')::int >= 75 THEN 'nice' ELSE 'ok' END)
            ELSE pick
        END)
    FROM jsonb_array_elements(sc.preferences->'picks') AS pick
), '[]'::jsonb))
WHERE jsonb_path_exists(sc.preferences, '$.picks[*] ? (@.optionId starts with "seniority:")');

DELETE FROM answer_corrections WHERE option_id IN ('seniority:lead_staff', 'seniority:principal_head');
DELETE FROM scoring_options WHERE id IN ('seniority:lead_staff', 'seniority:principal_head');
UPDATE scoring_options SET retired_at = NULL WHERE id = 'seniority:staff';
UPDATE scoring_options SET question = 'Is the role pitched at Junior or graduate level (not Mid, Senior or above)?' WHERE id = 'seniority:junior';
UPDATE scoring_options SET question = 'Is the role pitched at Mid level (not Junior, Senior, Staff, Principal, Lead or Head of)?' WHERE id = 'seniority:mid';
UPDATE scoring_options SET question = 'Is the role pitched at Senior level (not Staff, Principal, Lead or Head of)?' WHERE id = 'seniority:senior';
