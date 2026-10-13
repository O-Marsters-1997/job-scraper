-- name: CreateRecruiterContact :one
INSERT INTO recruiter_contacts (user_id, job_id, contacted_on, channel, outcome)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: ListRecruiterContactsForJob :many
SELECT * FROM recruiter_contacts
WHERE user_id = $1 AND job_id = $2
ORDER BY contacted_on DESC, created_at DESC;

-- name: GetRecruiterContactSummaries :many
SELECT
    job_id,
    COUNT(*)::int AS contact_count,
    MAX(contacted_on)::date AS last_contacted_on,
    (ARRAY_AGG(outcome ORDER BY CASE outcome
        WHEN 'submitted' THEN 4
        WHEN 'spoke' THEN 3
        WHEN 'rejected' THEN 2
        ELSE 1 END DESC))[1]::text AS best_outcome
FROM recruiter_contacts
WHERE user_id = $1 AND job_id = ANY($2::uuid[])
GROUP BY job_id;
