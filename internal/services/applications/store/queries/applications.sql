-- name: CreateApplication :one
INSERT INTO applications (user_id, job_id, status_id, applied_at, salary_info, chase_by)
SELECT p.user_id, p.job_id, p.status_id, p.applied_at, p.salary_info,
       (SELECT CURRENT_DATE + s.reply_window_days FROM application_statuses s WHERE s.id = p.status_id)
FROM (
    SELECT sqlc.arg('user_id')::uuid AS user_id, sqlc.arg('job_id')::uuid AS job_id,
           COALESCE(sqlc.narg('status_id')::uuid,
                    (SELECT d.id FROM application_statuses d
                     WHERE d.user_id = sqlc.arg('user_id')::uuid AND d.system_key = 'draft')) AS status_id,
           sqlc.narg('applied_at')::date AS applied_at, sqlc.narg('salary_info')::text AS salary_info
) p
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
    s.colour       AS status_colour,
    d.id           AS draft_id,
    COALESCE(d.status, '')     AS draft_status,
    d.outcome                  AS draft_outcome,
    COALESCE(d.last_error, '') AS draft_last_error
FROM applications a
JOIN jobs j ON a.job_id = j.id
LEFT JOIN application_statuses s ON a.status_id = s.id
LEFT JOIN LATERAL (
    SELECT c.id, c.status, c.outcome, c.last_error
    FROM tailored_cvs c
    WHERE c.user_id = a.user_id AND c.job_id = a.job_id
      AND (c.outcome = 'kept' OR c.outcome IS NULL)
    ORDER BY (c.outcome = 'kept') DESC NULLS LAST
    LIMIT 1
) d ON TRUE
WHERE a.user_id = $1
  AND (sqlc.narg('status_id')::uuid IS NULL OR a.status_id = sqlc.narg('status_id'))
  AND (NOT sqlc.arg('chase')::boolean OR a.chase_by IS NOT NULL)
ORDER BY
    CASE WHEN sqlc.arg('chase')::boolean THEN a.chase_by END ASC,
    a.updated_at DESC;

-- name: UpdateApplication :one
UPDATE applications a
SET status_id = $3, applied_at = $4, salary_info = $5, updated_at = NOW(),
    chase_by = CASE WHEN a.status_id IS DISTINCT FROM $3
                    THEN (SELECT CURRENT_DATE + s.reply_window_days FROM application_statuses s WHERE s.id = $3::uuid)
                    ELSE a.chase_by END
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

-- name: MarkApplicationApplied :one
WITH applied AS (
    SELECT s.id, s.reply_window_days FROM application_statuses s
    WHERE s.user_id = sqlc.arg(user_id)::uuid AND s.system_key = 'applied'
), target AS (
    SELECT a.id, a.status_id AS from_status_id FROM applications a
    WHERE a.user_id = sqlc.arg(user_id)::uuid AND a.job_id = sqlc.arg(job_id)::uuid
)
UPDATE applications a
SET status_id = applied.id,
    applied_at = COALESCE(a.applied_at, sqlc.arg(now)::timestamptz::date),
    chase_by = sqlc.arg(now)::timestamptz::date + applied.reply_window_days,
    updated_at = sqlc.arg(now)::timestamptz
FROM applied, target
WHERE a.id = target.id AND a.status_id IS DISTINCT FROM applied.id
RETURNING a.id, target.from_status_id, applied.id AS to_status_id;

-- name: IsAppliedStatus :one
SELECT EXISTS (
    SELECT 1 FROM application_statuses s
    WHERE s.id = sqlc.arg(id)::uuid AND s.system_key = 'applied'
);
