-- name: InsertScoreFeedback :one
INSERT INTO score_feedback (user_id, kind, reason, picks, model, snapshot)
VALUES (sqlc.arg(user_id), sqlc.arg(kind)::text, sqlc.arg(reason)::text, sqlc.arg(picks), sqlc.arg(model)::text, sqlc.arg(snapshot))
RETURNING *;

-- name: ListScoreFeedback :many
SELECT * FROM score_feedback
WHERE user_id = sqlc.arg(user_id)
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg(row_limit)::int;

-- name: ClearScoreFeedback :execrows
DELETE FROM score_feedback WHERE user_id = $1;
