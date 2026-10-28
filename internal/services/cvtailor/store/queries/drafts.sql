-- name: InsertDraft :one
INSERT INTO tailored_cvs (user_id, job_id, base_doc_id, base_tab_id, achievement_ids, skill_swaps, mode)
SELECT sqlc.arg(user_id)::uuid, j.id, sqlc.arg(base_doc_id)::text, sqlc.arg(base_tab_id)::text, sqlc.arg(achievement_ids)::uuid[], sqlc.arg(skill_swaps)::jsonb, sqlc.arg(mode)::text
FROM jobs j WHERE j.id = sqlc.arg(job_id)::uuid
RETURNING *;

-- name: GetDraft :one
SELECT * FROM tailored_cvs WHERE id = $1 AND user_id = $2;

-- name: ClaimDraft :one
WITH next AS (
    SELECT id FROM tailored_cvs
    WHERE (status = 'pending' AND due_at <= NOW())
       OR (status = 'running' AND lease_until <= NOW())
       OR (status = 'keeping' AND due_at <= NOW() AND (lease_until IS NULL OR lease_until <= NOW()))
    ORDER BY due_at, id LIMIT 1 FOR UPDATE SKIP LOCKED
)
UPDATE tailored_cvs t SET status = CASE WHEN t.status = 'keeping' THEN 'keeping' ELSE 'running' END, attempts = t.attempts + 1,
    lease_until = NOW() + interval '5 minutes'
FROM next, jobs j
LEFT JOIN companies c ON c.id = j.company_id
WHERE t.id = next.id AND j.id = t.job_id
RETURNING t.id, t.user_id, t.job_id, t.base_doc_id, t.base_tab_id, t.achievement_ids, t.skill_swaps, t.mode, t.attempts,
    COALESCE(t.draft_doc_id, '')::text AS draft_doc_id,
    t.status = 'keeping' AS keeping,
    j.title AS job_title,
    COALESCE(c.name, j.company_slug)::text AS company_name,
    j.description AS job_description,
    COALESCE(j.content_fingerprint, '')::text AS job_fingerprint;

-- name: SetDraftDoc :exec
UPDATE tailored_cvs SET draft_doc_id = sqlc.arg(draft_doc_id)::text
WHERE id = sqlc.arg(id)::uuid AND attempts = sqlc.arg(attempts)::int AND status = 'running';

-- name: CompleteDraft :execrows
UPDATE tailored_cvs SET status = 'ready', lease_until = NULL, last_error = '',
    edit_set = sqlc.arg(edit_set)::jsonb, base_content = sqlc.arg(base_content)::jsonb, raw_output = sqlc.arg(raw_output)::text,
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

-- name: QueueKeep :one
UPDATE tailored_cvs SET status = 'keeping', attempts = 0, due_at = NOW(), lease_until = NULL, last_error = ''
WHERE id = $1 AND user_id = $2 AND status = 'ready' AND outcome IS NULL
RETURNING *;

-- name: CompleteKeep :execrows
UPDATE tailored_cvs SET status = 'ready', outcome = 'kept', kept_as = 'doc', lease_until = NULL, last_error = ''
WHERE id = sqlc.arg(id)::uuid AND attempts = sqlc.arg(attempts)::int AND status = 'keeping';

-- name: FailKeep :execrows
UPDATE tailored_cvs SET
    status = CASE WHEN sqlc.arg(terminal)::bool OR attempts >= sqlc.arg(max_attempts)::int THEN 'ready' ELSE 'keeping' END,
    due_at = NOW() + make_interval(secs => LEAST(3600, 30 * power(2, attempts)::int)),
    lease_until = NULL,
    last_error = sqlc.arg(last_error)::text
WHERE id = sqlc.arg(id)::uuid AND attempts = sqlc.arg(attempts)::int AND status = 'keeping';

-- name: SetDraftEdits :execrows
UPDATE tailored_cvs SET edit_set = sqlc.arg(edit_set)::jsonb, findings = sqlc.arg(findings)::jsonb,
    achievement_ids = sqlc.arg(achievement_ids)::uuid[]
WHERE id = sqlc.arg(id)::uuid AND user_id = sqlc.arg(user_id)::uuid;

-- name: GetJobDescription :one
SELECT description FROM jobs WHERE id = $1;

-- name: SetDraftInsights :execrows
UPDATE tailored_cvs SET insights = sqlc.arg(insights)::jsonb, insights_for = sqlc.arg(insights_for)::text
WHERE id = sqlc.arg(id)::uuid AND user_id = sqlc.arg(user_id)::uuid;

-- name: GetJobNote :one
SELECT body FROM job_notes WHERE user_id = $1 AND job_id = $2;

-- name: ListJobCommentBodies :many
SELECT body FROM job_comments WHERE user_id = $1 AND job_id = $2 ORDER BY created_at, id LIMIT 200;

-- name: DeleteAchievementOutcomes :exec
DELETE FROM achievement_outcomes WHERE draft_id = $1 AND user_id = $2;

-- name: InsertAchievementOutcomes :copyfrom
INSERT INTO achievement_outcomes (draft_id, achievement_id, user_id, job_id, p_yes, p_no, on_source_cv, in_final_cv, outscored_kept_but_left_out, reworded)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10);

-- name: AddProfileAttempt :execrows
INSERT INTO profile_attempts (draft_id, text, steer, rejection, prompt_key, model, cost, created_at)
SELECT d.id, @text::text, @steer::text, @rejection::text, @prompt_key::text, @model::text, @cost::float8, clock_timestamp()
FROM tailored_cvs d
WHERE d.id = @draft_id::uuid AND d.user_id = @user_id::uuid;

-- name: ListProfileCaptureDrafts :many
SELECT d.id, d.base_content, d.edit_set, j.title AS job_title,
    COALESCE(c.name, j.company_slug)::text AS company, j.description AS job_description
FROM tailored_cvs d
JOIN jobs j ON j.id = d.job_id
LEFT JOIN companies c ON c.id = j.company_id
WHERE d.user_id = $1 AND d.outcome = 'kept'
  AND EXISTS (SELECT 1 FROM profile_attempts a WHERE a.draft_id = d.id)
ORDER BY d.created_at, d.id;

-- name: ListProfileAttempts :many
SELECT draft_id, text, steer, rejection, prompt_key, model, cost::float8 AS cost
FROM profile_attempts
WHERE draft_id = ANY($1::uuid[])
ORDER BY draft_id, created_at, id;
