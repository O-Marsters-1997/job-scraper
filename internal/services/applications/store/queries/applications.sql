-- name: CreateApplication :one
INSERT INTO applications (user_id, job_id, status_id, applied_at, salary_info)
SELECT sqlc.arg('user_id')::uuid, sqlc.arg('job_id')::uuid,
       COALESCE(sqlc.narg('status_id')::uuid,
                (SELECT d.id FROM application_statuses d
                 WHERE d.user_id = sqlc.arg('user_id')::uuid AND d.system_key = 'draft')),
       sqlc.narg('applied_at')::date, sqlc.narg('salary_info')::text
WHERE sqlc.narg('status_id')::uuid IS NULL
   OR EXISTS (SELECT 1 FROM application_statuses s
              WHERE s.id = sqlc.narg('status_id')::uuid AND s.user_id = sqlc.arg('user_id')::uuid)
RETURNING *;

-- name: ListApplications :many
SELECT
    a.id, a.user_id, a.job_id, a.status_id,
    a.applied_at, a.salary_info, a.created_at, a.updated_at, a.chase_by,
    j.title        AS job_title,
    j.company_slug AS job_company_slug,
    j.location     AS job_location,
    j.url          AS job_url,
    j.closed_at    AS job_closed_at,
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
UPDATE applications a
SET status_id = $3, applied_at = $4, salary_info = $5, updated_at = NOW(),
    chase_by = CASE WHEN a.status_id IS DISTINCT FROM $3 THEN NULL ELSE a.chase_by END
WHERE a.id = $1 AND a.user_id = $2
  AND ($3::uuid IS NULL
       OR EXISTS (SELECT 1 FROM application_statuses s WHERE s.id = $3::uuid AND s.user_id = $2))
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

-- name: GetApplicationState :one
SELECT job_id, status_id FROM applications
WHERE id = $1 AND user_id = $2;
