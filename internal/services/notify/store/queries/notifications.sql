-- name: InsertNotification :one
INSERT INTO notifications (user_id, job_id, title, body)
VALUES (sqlc.arg(user_id), sqlc.arg(job_id), sqlc.arg(title), sqlc.arg(body))
RETURNING id;

-- name: ListNotifications :many
SELECT n.id, n.kind, n.job_id, n.title, n.body, n.created_at, n.read_at,
    j.title AS job_title,
    COALESCE(c.name, j.company_slug)::text AS company_name,
    j.location, j.salary_raw,
    COALESCE(s.band, '')::text AS band,
    s.suitability_score
FROM notifications n
JOIN jobs j ON j.id = n.job_id
LEFT JOIN companies c ON c.id = j.company_id
LEFT JOIN job_scores s ON s.job_id = n.job_id AND s.user_id = n.user_id
WHERE n.user_id = sqlc.arg(user_id)
ORDER BY (n.read_at IS NOT NULL), n.created_at DESC
LIMIT 50;

-- name: GetNotificationJob :one
SELECT j.title,
    COALESCE(c.name, j.company_slug)::text AS company_name,
    j.location, j.salary_raw,
    COALESCE(s.band, '')::text AS band,
    s.suitability_score
FROM jobs j
LEFT JOIN companies c ON c.id = j.company_id
LEFT JOIN job_scores s ON s.job_id = j.id AND s.user_id = sqlc.arg(user_id)
WHERE j.id = sqlc.arg(job_id);

-- name: MarkNotificationRead :exec
UPDATE notifications SET read_at = NOW()
WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id) AND read_at IS NULL;

-- name: MarkAllNotificationsRead :exec
UPDATE notifications SET read_at = NOW()
WHERE user_id = sqlc.arg(user_id) AND read_at IS NULL;
