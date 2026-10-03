-- name: UpsertGrade :one
INSERT INTO job_grades (user_id, job_id, grade, reasons, score_at_grade, score_model)
VALUES (sqlc.arg(user_id), sqlc.arg(job_id), sqlc.arg(grade)::text, sqlc.arg(reasons)::text[], sqlc.narg(score_at_grade), sqlc.narg(score_model))
ON CONFLICT (user_id, job_id) DO UPDATE
SET grade = EXCLUDED.grade, reasons = EXCLUDED.reasons, score_at_grade = EXCLUDED.score_at_grade,
    score_model = EXCLUDED.score_model, updated_at = NOW()
RETURNING *;

-- name: GetGrade :one
SELECT * FROM job_grades WHERE user_id = $1 AND job_id = $2;

-- name: DeleteGrade :execrows
DELETE FROM job_grades WHERE user_id = $1 AND job_id = $2;

-- name: ListGrades :many
SELECT * FROM job_grades WHERE user_id = $1 ORDER BY updated_at DESC, job_id;

-- name: ListImpliedPositives :many
SELECT job_id, 'application'::text AS source FROM applications WHERE applications.user_id = $1
UNION ALL
SELECT job_id, 'kept_cv'::text AS source FROM tailored_cvs WHERE tailored_cvs.user_id = $1 AND outcome = 'kept'
ORDER BY source, job_id;
