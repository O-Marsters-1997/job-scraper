-- name: CreateApplicationStatus :one
INSERT INTO application_statuses (user_id, name, colour, reply_window_days)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: SeedDefaultStatuses :exec
INSERT INTO application_statuses (user_id, name, colour)
VALUES
    ($1, 'Draft',        '#64748b'),
    ($1, 'Applied',      '#6366f1'),
    ($1, 'Interviewing', '#f59e0b'),
    ($1, 'Rejected',     '#ef4444'),
    ($1, 'Offer',        '#22c55e');

-- name: ListApplicationStatusesByUser :many
SELECT * FROM application_statuses
WHERE user_id = $1
ORDER BY created_at ASC;

-- name: UpdateApplicationStatus :one
UPDATE application_statuses
SET name = $3, colour = $4, reply_window_days = $5
WHERE id = $1 AND user_id = $2
RETURNING *;

-- name: DeleteApplicationStatus :exec
DELETE FROM application_statuses
WHERE id = $1 AND user_id = $2;

-- name: CountApplicationsUsingStatus :one
SELECT COUNT(*) FROM applications
WHERE status_id = $1 AND user_id = $2;
