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
