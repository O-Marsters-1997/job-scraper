-- name: CreateRecruiterContact :one
INSERT INTO recruiter_contacts (user_id, job_id, contacted_on, channel, outcome)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: ListRecruiterContactsForJob :many
SELECT * FROM recruiter_contacts
WHERE user_id = $1 AND job_id = $2
ORDER BY contacted_on DESC, created_at DESC;
