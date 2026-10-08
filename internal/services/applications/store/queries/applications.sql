-- name: CreateApplication :one
INSERT INTO applications (user_id, job_id, status_id, notes, applied_at, salary_info)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: ListApplications :many
SELECT
    a.id, a.user_id, a.job_id, a.status_id,
    a.notes, a.applied_at, a.salary_info, a.created_at, a.updated_at, a.chase_by,
    j.title        AS job_title,
    j.company_slug AS job_company_slug,
    j.location     AS job_location,
    j.url          AS job_url,
    s.name         AS status_name,
    s.colour       AS status_colour
FROM applications a
JOIN jobs j ON a.job_id = j.id
LEFT JOIN application_statuses s ON a.status_id = s.id
WHERE a.user_id = $1
  AND (sqlc.narg('status_id')::uuid IS NULL OR a.status_id = sqlc.narg('status_id'))
  AND (NOT sqlc.arg('chase')::boolean OR a.chase_by IS NOT NULL)
ORDER BY
    CASE WHEN sqlc.arg('chase')::boolean THEN a.chase_by END ASC,
    a.updated_at DESC;

-- name: UpdateApplication :one
UPDATE applications
SET status_id = $3, notes = $4, applied_at = $5, salary_info = $6, updated_at = NOW(),
    chase_by = CASE WHEN status_id IS DISTINCT FROM $3 THEN NULL ELSE chase_by END
WHERE id = $1 AND user_id = $2
RETURNING *;

-- name: SetApplicationChase :one
UPDATE applications
SET chase_by = $3
WHERE id = $1 AND user_id = $2
RETURNING *;

-- name: ClearApplicationChase :exec
UPDATE applications
SET chase_by = NULL
WHERE id = $1 AND user_id = $2;

-- name: DeleteApplication :exec
DELETE FROM applications
WHERE id = $1 AND user_id = $2;

-- name: GetApplicationsForJobs :many
SELECT
    a.job_id,
    a.id,
    a.status_id,
    s.name   AS status_name,
    s.colour AS status_colour
FROM applications a
LEFT JOIN application_statuses s ON a.status_id = s.id
WHERE a.user_id = $1 AND a.job_id = ANY($2::uuid[]);
