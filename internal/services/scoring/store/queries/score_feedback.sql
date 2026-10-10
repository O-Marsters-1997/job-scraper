-- name: InsertScoreFeedback :one
INSERT INTO score_feedback (user_id, job_id, kind, direction, reason, picks, model, snapshot)
VALUES (sqlc.arg(user_id), sqlc.narg(job_id), sqlc.arg(kind)::text, sqlc.narg(direction)::text, sqlc.arg(reason)::text, sqlc.arg(picks), sqlc.arg(model)::text, sqlc.arg(snapshot))
RETURNING *;

-- name: ListScoreFeedback :many
SELECT sf.*, d.picks_changed, d.model_changed
FROM score_feedback sf
CROSS JOIN LATERAL (
    SELECT sf.picks IS DISTINCT FROM COALESCE(NULLIF((SELECT preferences->'picks' FROM search_config WHERE user_id = sf.user_id), 'null'::jsonb), '[]'::jsonb) AS picks_changed,
           sf.model <> sqlc.arg(model)::text AS model_changed
) d
WHERE sf.user_id = sqlc.arg(user_id)
  AND (sqlc.arg(kind)::text = '' OR sf.kind = sqlc.arg(kind)::text)
  AND (sqlc.arg(include_outdated)::bool OR NOT (d.picks_changed OR d.model_changed))
ORDER BY sf.created_at DESC, sf.id DESC
LIMIT sqlc.arg(row_limit)::int OFFSET sqlc.arg(row_offset)::int;

-- name: CountScoreFeedback :one
SELECT count(*) FILTER (WHERE NOT (d.picks_changed OR d.model_changed))::bigint AS current,
       count(*) FILTER (WHERE d.picks_changed OR d.model_changed)::bigint AS outdated
FROM score_feedback sf
CROSS JOIN LATERAL (
    SELECT sf.picks IS DISTINCT FROM COALESCE(NULLIF((SELECT preferences->'picks' FROM search_config WHERE user_id = sf.user_id), 'null'::jsonb), '[]'::jsonb) AS picks_changed,
           sf.model <> sqlc.arg(model)::text AS model_changed
) d
WHERE sf.user_id = sqlc.arg(user_id)
  AND (sqlc.arg(kind)::text = '' OR sf.kind = sqlc.arg(kind)::text);

-- name: DeleteScoreFeedback :execrows
DELETE FROM score_feedback WHERE id = $1 AND user_id = $2;

-- name: ClearScoreFeedback :execrows
DELETE FROM score_feedback WHERE user_id = $1;

-- name: GetJobScoreForFeedback :one
SELECT suitability_score::int AS score, COALESCE(band, '')::text AS band, breakdown, COALESCE(score_fingerprint, '')::text AS score_fingerprint,
    COALESCE(score_model, '')::text AS score_model
FROM job_scores
WHERE user_id = sqlc.arg(user_id) AND job_id = sqlc.arg(job_id) AND suitability_score IS NOT NULL;

-- name: ListJobScoresForCollection :many
SELECT j.id, j.title, j.company_slug, js.suitability_score AS score, COALESCE(js.breakdown, '[]'::jsonb) AS breakdown
FROM jobs j
LEFT JOIN job_scores js ON js.job_id = j.id AND js.user_id = sqlc.arg(user_id)
WHERE j.id = ANY(sqlc.arg(job_ids)::uuid[]);
