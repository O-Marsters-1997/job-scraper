-- name: InsertNotificationDigest :one
INSERT INTO notification_digests (sent_at, job_count)
VALUES ($1, $2)
RETURNING *;

-- name: GetLastNotificationDigestSentAt :one
SELECT sent_at FROM notification_digests ORDER BY sent_at DESC LIMIT 1;
