-- name: InsertNotification :one
INSERT INTO notifications (user_id, job_id, title, body)
VALUES (sqlc.arg(user_id), sqlc.arg(job_id), sqlc.arg(title), sqlc.arg(body))
RETURNING id;
