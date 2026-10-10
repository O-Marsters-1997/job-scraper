-- name: InsertEvent :exec
INSERT INTO events (user_id, type, subject_id, props)
VALUES (sqlc.arg(user_id), sqlc.arg(type), sqlc.narg(subject_id), sqlc.arg(props));

-- name: ListEventsByUser :many
SELECT id, type, subject_id, props, created_at
FROM events
WHERE user_id = sqlc.arg(user_id)
  AND (sqlc.narg(type)::event_type IS NULL OR type = sqlc.narg(type))
ORDER BY created_at, id;
