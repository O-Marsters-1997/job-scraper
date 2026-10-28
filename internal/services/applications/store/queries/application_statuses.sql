-- name: CreateApplicationStatus :one
INSERT INTO application_statuses (user_id, name, colour, reply_window_days, counts_as_reply)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: SeedDefaultStatuses :exec
INSERT INTO application_statuses (user_id, name, colour, system_key, counts_as_reply)
VALUES
    ($1, 'Draft',        '#64748b', 'draft',   FALSE),
    ($1, 'Applied',      '#6366f1', 'applied', FALSE),
    ($1, 'Interviewing', '#f59e0b', NULL, TRUE),
    ($1, 'Rejected',     '#ef4444', NULL, FALSE),
    ($1, 'Offer',        '#22c55e', NULL, TRUE);

-- name: ListApplicationStatusesByUser :many
SELECT * FROM application_statuses
WHERE user_id = $1
ORDER BY created_at ASC;

-- name: UpdateApplicationStatus :one
UPDATE application_statuses
SET name = $3, colour = $4, reply_window_days = $5, counts_as_reply = $6
WHERE id = $1 AND user_id = $2
RETURNING *;

-- name: DeleteApplicationStatus :exec
DELETE FROM application_statuses
WHERE id = $1 AND user_id = $2;

-- name: CountApplicationsUsingStatus :one
SELECT COUNT(*) FROM applications
WHERE status_id = $1 AND user_id = $2;
