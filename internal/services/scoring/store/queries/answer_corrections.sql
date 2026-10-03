-- name: UpsertAnswerCorrection :exec
INSERT INTO answer_corrections (user_id, job_id, option_id, value)
VALUES (sqlc.arg(user_id)::uuid, sqlc.arg(job_id)::uuid, sqlc.arg(option_id)::text, sqlc.arg(value)::text)
ON CONFLICT (user_id, job_id, option_id) DO UPDATE SET value = EXCLUDED.value;

-- name: DeleteAnswerCorrection :exec
DELETE FROM answer_corrections
WHERE user_id = sqlc.arg(user_id)::uuid AND job_id = sqlc.arg(job_id)::uuid AND option_id = sqlc.arg(option_id)::text;

-- name: ListAnswerCorrectionsForJob :many
SELECT user_id, option_id, value FROM answer_corrections WHERE job_id = sqlc.arg(job_id)::uuid;

-- name: ListAnswerCorrectionsForUser :many
SELECT job_id, option_id, value FROM answer_corrections WHERE user_id = sqlc.arg(user_id)::uuid;
