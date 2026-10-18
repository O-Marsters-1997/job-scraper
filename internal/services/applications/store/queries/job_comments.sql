-- name: CreateJobComment :one
INSERT INTO job_comments (user_id, job_id, body, quote, quote_prefix, quote_suffix)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, user_id, job_id, body, quote, quote_prefix, quote_suffix, created_at, updated_at;

-- name: ListJobCommentsForJob :many
SELECT id, user_id, job_id, body, quote, quote_prefix, quote_suffix, created_at, updated_at FROM job_comments
WHERE user_id = $1 AND job_id = $2
ORDER BY created_at DESC, id DESC
LIMIT 1000;

-- name: UpdateJobCommentBody :one
UPDATE job_comments SET body = $3, updated_at = now()
WHERE id = $1 AND user_id = $2
RETURNING id, user_id, job_id, body, quote, quote_prefix, quote_suffix, created_at, updated_at;

-- name: DeleteJobComment :execrows
DELETE FROM job_comments WHERE id = $1 AND user_id = $2;
