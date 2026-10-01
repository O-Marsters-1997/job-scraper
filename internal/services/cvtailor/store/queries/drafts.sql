-- name: InsertDraft :one
INSERT INTO tailored_cvs (user_id, job_id, base_doc_id, base_tab_id, achievement_ids)
SELECT sqlc.arg(user_id)::uuid, j.id, sqlc.arg(base_doc_id)::text, sqlc.arg(base_tab_id)::text, sqlc.arg(achievement_ids)::uuid[]
FROM jobs j WHERE j.id = sqlc.arg(job_id)::uuid
RETURNING *;

-- name: GetDraft :one
SELECT * FROM tailored_cvs WHERE id = $1 AND user_id = $2;

-- name: ClaimDraft :one
WITH next AS (
    SELECT id FROM tailored_cvs
    WHERE (status = 'pending' AND due_at <= NOW())
       OR (status = 'running' AND lease_until <= NOW())
    ORDER BY due_at, id LIMIT 1 FOR UPDATE SKIP LOCKED
)
UPDATE tailored_cvs t SET status = 'running', attempts = t.attempts + 1,
    lease_until = NOW() + interval '5 minutes'
FROM next, jobs j
WHERE t.id = next.id AND j.id = t.job_id
RETURNING t.id, t.user_id, t.job_id, t.base_doc_id, t.base_tab_id, t.achievement_ids, t.attempts,
    COALESCE(t.draft_doc_id, '')::text AS draft_doc_id,
    j.description AS job_description,
    COALESCE(j.content_fingerprint, '')::text AS job_fingerprint;

-- name: SetDraftDoc :exec
UPDATE tailored_cvs SET draft_doc_id = sqlc.arg(draft_doc_id)::text
WHERE id = sqlc.arg(id)::uuid AND attempts = sqlc.arg(attempts)::int AND status = 'running';

-- name: CompleteDraft :execrows
UPDATE tailored_cvs SET status = 'ready', lease_until = NULL, last_error = '',
    edit_set = sqlc.arg(edit_set)::jsonb, raw_output = sqlc.arg(raw_output)::text,
    model = sqlc.arg(model)::text, prompt_version = sqlc.arg(prompt_version)::text,
    job_fingerprint = sqlc.arg(job_fingerprint)::text, cost = sqlc.arg(cost)::real,
    draft_doc_id = sqlc.arg(draft_doc_id)::text, findings = sqlc.arg(findings)::jsonb
WHERE id = sqlc.arg(id)::uuid AND attempts = sqlc.arg(attempts)::int AND status = 'running';

-- name: FailDraft :execrows
UPDATE tailored_cvs SET
    status = CASE WHEN sqlc.arg(terminal)::bool OR attempts >= sqlc.arg(max_attempts)::int THEN 'failed' ELSE 'pending' END,
    due_at = NOW() + make_interval(secs => LEAST(3600, 30 * power(2, attempts)::int)),
    lease_until = NULL,
    last_error = sqlc.arg(last_error)::text,
    draft_doc_id = CASE WHEN sqlc.arg(clear_doc)::bool THEN NULL ELSE draft_doc_id END
WHERE id = sqlc.arg(id)::uuid AND attempts = sqlc.arg(attempts)::int AND status = 'running';

-- name: ListJobDrafts :many
SELECT * FROM tailored_cvs WHERE job_id = $1 AND user_id = $2 ORDER BY created_at DESC, id;

-- name: SetDraftOutcome :one
UPDATE tailored_cvs SET outcome = sqlc.arg(outcome)::text,
    draft_doc_id = CASE WHEN sqlc.arg(outcome)::text = 'discarded' THEN NULL ELSE draft_doc_id END
WHERE id = sqlc.arg(id)::uuid AND user_id = sqlc.arg(user_id)::uuid
RETURNING *;

-- name: SetDraftEdits :execrows
UPDATE tailored_cvs SET edit_set = sqlc.arg(edit_set)::jsonb, findings = sqlc.arg(findings)::jsonb
WHERE id = sqlc.arg(id)::uuid AND user_id = sqlc.arg(user_id)::uuid;
