-- name: ListPromptPartOverrides :many
SELECT path, text, base_default
FROM prompt_part_overrides
WHERE user_id = $1 AND generation = $2
ORDER BY path
LIMIT 100;

-- name: UpsertPromptPartOverride :exec
INSERT INTO prompt_part_overrides (user_id, generation, path, text, base_default)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (user_id, generation, path)
DO UPDATE SET text = EXCLUDED.text, base_default = EXCLUDED.base_default, updated_at = now();

-- name: DeletePromptPartOverride :exec
DELETE FROM prompt_part_overrides
WHERE user_id = $1 AND generation = $2 AND path = $3;
