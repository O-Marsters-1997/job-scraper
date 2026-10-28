-- name: GetJobNote :one
SELECT id, user_id, job_id, body, created_at, updated_at FROM job_notes
WHERE user_id = $1 AND job_id = $2;

-- name: UpsertJobNote :one
INSERT INTO job_notes (user_id, job_id, body)
VALUES ($1, $2, $3)
ON CONFLICT (user_id, job_id) DO UPDATE SET body = EXCLUDED.body, updated_at = now()
RETURNING id, user_id, job_id, body, created_at, updated_at;
