-- name: ListScoringOptions :many
SELECT * FROM scoring_options
WHERE owner_user_id IS NULL OR (owner_user_id = sqlc.arg(user_id)::uuid AND retired_at IS NULL)
ORDER BY dimension, id;

-- name: InsertScoringOption :exec
INSERT INTO scoring_options (id, dimension, label, question)
VALUES (sqlc.arg(id)::text, sqlc.arg(dimension)::scoring_dimension, sqlc.arg(label)::text, sqlc.arg(question)::text);

-- name: RewordScoringOption :execrows
UPDATE scoring_options SET question = sqlc.arg(question)::text WHERE id = sqlc.arg(id)::text;

-- name: RetireScoringOption :execrows
UPDATE scoring_options SET retired_at = NOW() WHERE id = $1 AND retired_at IS NULL;

-- name: InsertCustomOption :exec
INSERT INTO scoring_options (id, dimension, label, question, owner_user_id)
VALUES (sqlc.arg(id)::text, sqlc.arg(dimension)::scoring_dimension, sqlc.arg(label)::text, sqlc.arg(question)::text, sqlc.arg(user_id)::uuid);

-- name: RetireCustomOption :execrows
UPDATE scoring_options SET retired_at = NOW()
WHERE id = sqlc.arg(id)::text AND owner_user_id = sqlc.arg(user_id)::uuid AND retired_at IS NULL;

-- name: CountLiveCustomOptions :one
SELECT COUNT(*) FROM scoring_options WHERE owner_user_id = sqlc.arg(user_id)::uuid AND retired_at IS NULL;

-- name: MovePick :exec
UPDATE search_config SET
    preferences = jsonb_set(preferences, '{picks}', (
        SELECT COALESCE(jsonb_agg(
            CASE WHEN t.pick ->> 'optionId' = sqlc.arg(old_id)::text
                THEN jsonb_set(t.pick, '{optionId}', to_jsonb(sqlc.arg(new_id)::text))
                ELSE t.pick END
            ORDER BY t.ord), '[]'::jsonb)
        FROM jsonb_array_elements(preferences -> 'picks') WITH ORDINALITY AS t(pick, ord)
    )),
    updated_at = NOW()
WHERE user_id = sqlc.arg(user_id)::uuid
    AND jsonb_typeof(preferences -> 'picks') = 'array'
    AND EXISTS (
        SELECT 1 FROM jsonb_array_elements(preferences -> 'picks') AS p
        WHERE p ->> 'optionId' = sqlc.arg(old_id)::text
    );

-- name: DropPick :exec
UPDATE search_config SET
    preferences = jsonb_set(preferences, '{picks}', (
        SELECT COALESCE(jsonb_agg(t.pick ORDER BY t.ord), '[]'::jsonb)
        FROM jsonb_array_elements(preferences -> 'picks') WITH ORDINALITY AS t(pick, ord)
        WHERE t.pick ->> 'optionId' <> sqlc.arg(option_id)::text
    )),
    updated_at = NOW()
WHERE user_id = sqlc.arg(user_id)::uuid
    AND jsonb_typeof(preferences -> 'picks') = 'array'
    AND EXISTS (
        SELECT 1 FROM jsonb_array_elements(preferences -> 'picks') AS p
        WHERE p ->> 'optionId' = sqlc.arg(option_id)::text
    );
