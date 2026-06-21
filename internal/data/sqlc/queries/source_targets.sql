-- name: ListSourceTargetsByUser :many
SELECT * FROM source_targets WHERE user_id = $1 ORDER BY source, value;

-- name: ListEnabledSourceTargets :many
SELECT * FROM source_targets WHERE enabled = TRUE ORDER BY source, value;

-- name: CreateSourceTarget :one
INSERT INTO source_targets (user_id, source, value, enabled, filters)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: UpdateSourceTarget :one
UPDATE source_targets SET enabled = $3, updated_at = NOW()
WHERE id = $1 AND user_id = $2
RETURNING *;

-- name: DeleteSourceTarget :exec
DELETE FROM source_targets WHERE id = $1 AND user_id = $2;
