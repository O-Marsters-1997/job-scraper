-- name: CreateJobNote :one
INSERT INTO job_notes (user_id, job_id, body)
VALUES ($1, $2, $3)
RETURNING id, user_id, job_id, body, created_at, updated_at;

-- name: ListJobNotesForJob :many
SELECT id, user_id, job_id, body, created_at, updated_at FROM job_notes
WHERE user_id = $1 AND job_id = $2
ORDER BY created_at DESC, id DESC
LIMIT 1000;
