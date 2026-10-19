-- name: InsertCoverLetter :one
INSERT INTO cover_letters (user_id, job_id, source_doc_id, source_tab_id, scaffold_tab_id, achievement_ids, model, cost)
SELECT sqlc.arg(user_id)::uuid, j.id, sqlc.arg(source_doc_id)::text, sqlc.arg(source_tab_id)::text,
    sqlc.arg(scaffold_tab_id)::text, sqlc.arg(achievement_ids)::uuid[], sqlc.arg(model)::text, sqlc.arg(cost)::real
FROM jobs j WHERE j.id = sqlc.arg(job_id)::uuid
RETURNING id, user_id, job_id, source_doc_id, source_tab_id, scaffold_tab_id, achievement_ids, model, cost, created_at;

-- name: ListJobCoverLetters :many
SELECT id, user_id, job_id, source_doc_id, source_tab_id, scaffold_tab_id, achievement_ids, model, cost, created_at
FROM cover_letters WHERE job_id = $1 AND user_id = $2
ORDER BY created_at DESC, id
LIMIT 100;

-- name: IsLetterTab :one
SELECT EXISTS (
    SELECT 1 FROM tracked_docs d
    JOIN tracked_doc_tabs t ON t.tracked_doc_id = d.id
    WHERE d.user_id = $1 AND d.doc_id = $2 AND t.tab_id = $3 AND d.kind = 'letter'
);

-- name: GetJobBrief :one
SELECT j.title, COALESCE(c.name, j.company_slug)::text AS company_name, j.description
FROM jobs j
LEFT JOIN companies c ON c.id = j.company_id
WHERE j.id = $1;
