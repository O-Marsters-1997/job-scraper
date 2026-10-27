-- name: InsertNotification :one
INSERT INTO notifications (user_id, job_id, title, body)
VALUES (sqlc.arg(user_id), sqlc.arg(job_id), sqlc.arg(title), sqlc.arg(body))
RETURNING id;


-- name: ListNotifications :many
SELECT id, job_id, title, body, created_at, read_at
FROM notifications
WHERE user_id = sqlc.arg(user_id)
ORDER BY (read_at IS NOT NULL), created_at DESC
LIMIT 50;

-- name: MarkNotificationRead :exec
UPDATE notifications SET read_at = NOW()
WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id) AND read_at IS NULL;

-- name: MarkAllNotificationsRead :exec
UPDATE notifications SET read_at = NOW()
WHERE user_id = sqlc.arg(user_id) AND read_at IS NULL;
