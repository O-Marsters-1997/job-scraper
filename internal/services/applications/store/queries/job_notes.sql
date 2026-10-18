-- name: CreateJobNote :one
INSERT INTO job_notes (user_id, job_id, body)
VALUES ($1, $2, $3)
RETURNING id, user_id, job_id, body, created_at, updated_at;

-- name: ListJobNotesForJob :many
SELECT id, user_id, job_id, body, created_at, updated_at FROM job_notes
WHERE user_id = $1 AND job_id = $2
ORDER BY created_at DESC, id DESC
LIMIT 1000;

-- name: UpdateJobNoteBody :one
UPDATE job_notes SET body = $3, updated_at = now()
WHERE id = $1 AND user_id = $2
RETURNING id, user_id, job_id, body, created_at, updated_at;

-- name: DeleteJobNote :execrows
DELETE FROM job_notes WHERE id = $1 AND user_id = $2;
