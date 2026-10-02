-- name: InsertScoreFeedback :one
INSERT INTO score_feedback (user_id, job_id, kind, direction, reason, picks, model, snapshot)
VALUES (sqlc.arg(user_id), sqlc.narg(job_id), sqlc.arg(kind)::text, sqlc.narg(direction)::text, sqlc.arg(reason)::text, sqlc.arg(picks), sqlc.arg(model)::text, sqlc.arg(snapshot))
RETURNING *;

-- name: ListScoreFeedback :many
SELECT * FROM score_feedback
WHERE user_id = sqlc.arg(user_id)
  AND (sqlc.arg(kind)::text = '' OR kind = sqlc.arg(kind)::text)
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg(row_limit)::int OFFSET sqlc.arg(row_offset)::int;

-- name: CountScoreFeedback :one
SELECT count(*) FROM score_feedback
WHERE user_id = sqlc.arg(user_id)
  AND (sqlc.arg(kind)::text = '' OR kind = sqlc.arg(kind)::text);

-- name: DeleteScoreFeedback :execrows
DELETE FROM score_feedback WHERE id = $1 AND user_id = $2;

-- name: ClearScoreFeedback :execrows
DELETE FROM score_feedback WHERE user_id = $1;

-- name: GetJobScoreForFeedback :one
SELECT suitability_score::int AS score, breakdown, COALESCE(score_fingerprint, '')::text AS score_fingerprint,
    COALESCE(score_model, '')::text AS score_model
FROM job_scores
WHERE user_id = sqlc.arg(user_id) AND job_id = sqlc.arg(job_id) AND suitability_score IS NOT NULL;
