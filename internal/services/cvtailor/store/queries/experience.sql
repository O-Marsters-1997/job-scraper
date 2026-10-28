-- name: ListPositions :many
SELECT * FROM positions
WHERE user_id = $1
ORDER BY sort_order, created_at;

-- name: ListAchievements :many
SELECT * FROM achievements
WHERE user_id = $1
ORDER BY sort_order, created_at;

-- name: CreatePosition :one
INSERT INTO positions (user_id, employer, title, start_date, end_date, sort_order)
VALUES (
    sqlc.arg(user_id), sqlc.arg(employer), sqlc.arg(title), sqlc.arg(start_date), sqlc.arg(end_date),
    (SELECT COALESCE(MIN(p.sort_order), 1) - 1 FROM positions p WHERE p.user_id = sqlc.arg(user_id))
)
RETURNING *;

-- name: UpdatePosition :one
UPDATE positions
SET employer = sqlc.arg(employer), title = sqlc.arg(title),
    start_date = sqlc.arg(start_date), end_date = sqlc.arg(end_date), updated_at = NOW()
WHERE user_id = sqlc.arg(user_id) AND id = sqlc.arg(id)
RETURNING *;

-- name: DeletePosition :execrows
DELETE FROM positions WHERE user_id = $1 AND id = $2;

-- name: ReorderPositions :execrows
UPDATE positions p
SET sort_order = o.ord::int
FROM unnest(sqlc.arg(ids)::uuid[]) WITH ORDINALITY AS o(id, ord)
WHERE p.user_id = sqlc.arg(user_id) AND p.id = o.id;

-- name: CreateAchievement :one
INSERT INTO achievements (position_id, user_id, text, sort_order)
SELECT p.id, p.user_id, sqlc.arg(text),
       (SELECT COALESCE(MAX(a.sort_order), 0) + 1 FROM achievements a WHERE a.position_id = p.id)
FROM positions p
WHERE p.user_id = sqlc.arg(user_id) AND p.id = sqlc.arg(position_id)
RETURNING *;

-- name: UpdateAchievement :one
UPDATE achievements
SET text = sqlc.arg(text), updated_at = NOW()
WHERE user_id = sqlc.arg(user_id) AND id = sqlc.arg(id)
RETURNING *;

-- name: DeleteAchievement :execrows
DELETE FROM achievements WHERE user_id = $1 AND id = $2;

-- name: ReorderAchievements :execrows
UPDATE achievements a
SET sort_order = o.ord::int
FROM unnest(sqlc.arg(ids)::uuid[]) WITH ORDINALITY AS o(id, ord)
WHERE a.user_id = sqlc.arg(user_id) AND a.position_id = sqlc.arg(position_id) AND a.id = o.id;

-- name: GetPosition :one
SELECT * FROM positions WHERE user_id = $1 AND id = $2;

-- name: SetAchievementContexts :execrows
UPDATE achievements a
SET context = (sqlc.arg(contexts)::text[])[o.ord], updated_at = NOW()
FROM unnest(sqlc.arg(ids)::uuid[]) WITH ORDINALITY AS o(id, ord)
WHERE a.user_id = sqlc.arg(user_id) AND a.id = o.id;

-- name: ListAchievementStats :many
WITH reached AS (
    SELECT a.id AS application_id, a.status_id
    FROM applications a
    WHERE a.user_id = sqlc.arg(user_id)
    UNION
    SELECT e.subject_id, (e.props->>'to_status_id')::uuid
    FROM events e
    WHERE e.user_id = sqlc.arg(user_id)
      AND e.type = 'application_status_changed'
      AND e.props->>'to_status_id' IS NOT NULL
),
progress AS (
    SELECT r.application_id,
           COALESCE(bool_or(s.system_key = 'applied'), FALSE) AS applied,
           COALESCE(bool_or(s.counts_as_reply), FALSE) AS replied
    FROM reached r
    JOIN application_statuses s ON s.id = r.status_id
    GROUP BY r.application_id
)
SELECT o.achievement_id,
       (COUNT(*) FILTER (WHERE o.in_final_cv))::int AS in_cvs,
       (COUNT(*) FILTER (WHERE o.outscored_kept_but_left_out))::int AS left_out,
       (COUNT(*) FILTER (WHERE o.in_final_cv AND p.applied))::int AS applied,
       (COUNT(*) FILTER (WHERE o.in_final_cv AND p.applied AND p.replied))::int AS replied
FROM achievement_outcomes o
JOIN tailored_cvs t ON t.id = o.draft_id AND t.outcome = 'kept'
LEFT JOIN applications ap ON ap.user_id = o.user_id AND ap.job_id = o.job_id
LEFT JOIN progress p ON p.application_id = ap.id
WHERE o.user_id = sqlc.arg(user_id)
GROUP BY o.achievement_id;
