-- name: QueueAnswerEffect :exec
INSERT INTO effect_outbox (job_id, fingerprint, first_discovery)
SELECT sqlc.arg(job_id)::uuid, sqlc.arg(fingerprint)::text, sqlc.arg(first_discovery)::boolean
FROM jobs j
WHERE j.id = sqlc.arg(job_id)::uuid
    AND (
        EXISTS (
            SELECT 1 FROM tracked_companies tc JOIN companies c ON c.id = tc.company_id
            WHERE tc.enabled AND (c.id = j.company_id OR c.slug = j.company_slug)
        )
        OR EXISTS (
            SELECT 1 FROM source_targets st
            WHERE st.enabled AND st.source = j.source
                AND (sqlc.arg(discovery)::boolean OR st.value = j.company_slug)
        )
    )
ON CONFLICT (job_id, fingerprint, model) WHERE status IN ('pending', 'running') DO NOTHING;

-- name: QueueTrackingScores :exec
INSERT INTO effect_outbox (job_id, fingerprint)
SELECT j.id, j.content_fingerprint
FROM jobs j JOIN companies c ON c.id = sqlc.arg(company_id)::uuid
WHERE j.closed_at IS NULL AND j.content_fingerprint IS NOT NULL
    AND (j.company_id = c.id OR j.company_slug = c.slug)
ON CONFLICT (job_id, fingerprint, model) WHERE status IN ('pending', 'running') DO NOTHING;

-- name: QueueUserBackfill :execrows
INSERT INTO effect_outbox (job_id, fingerprint)
SELECT j.id, j.content_fingerprint
FROM job_scores s JOIN jobs j ON j.id = s.job_id
WHERE s.user_id = sqlc.arg(user_id)::uuid
    AND j.closed_at IS NULL AND j.content_fingerprint IS NOT NULL
ON CONFLICT (job_id, fingerprint, model) WHERE status IN ('pending', 'running') DO NOTHING;

-- name: BackfillCompanyJobFingerprints :exec
UPDATE jobs j SET content_fingerprint = encode(sha256(convert_to(
    replace(replace(replace(replace(replace(to_json(ARRAY[
        j.title, j.description, j.location, j.salary_raw, j.work_arrangement
    ])::text,
    '&', chr(92) || 'u0026'), '<', chr(92) || 'u003c'),
    '>', chr(92) || 'u003e'), chr(8232), chr(92) || 'u2028'),
    chr(8233), chr(92) || 'u2029'), 'UTF8')), 'hex')
FROM companies c
WHERE c.id = $1::uuid AND (j.company_id = c.id OR j.company_slug = c.slug)
    AND j.closed_at IS NULL AND j.content_fingerprint IS NULL;

-- name: ClaimAnswerEffect :one
WITH next AS (
    SELECT id FROM effect_outbox
    WHERE (status = 'pending' AND due_at <= NOW())
       OR (status = 'running' AND lease_until <= NOW())
    ORDER BY due_at, id LIMIT 1 FOR UPDATE SKIP LOCKED
)
UPDATE effect_outbox e SET status = 'running', attempts = attempts + 1,
    lease_until = NOW() + interval '5 minutes'
FROM next WHERE e.id = next.id
RETURNING e.id, e.job_id, e.fingerprint, e.model, e.attempts, e.first_discovery;

-- name: FailAnswerEffect :exec
UPDATE effect_outbox SET
    status = CASE WHEN sqlc.arg(terminal)::bool OR attempts >= 8 THEN 'failed' ELSE 'pending' END,
    due_at = NOW() + make_interval(secs =>
        COALESCE(sqlc.narg(retry_after_secs)::int, LEAST(3600, 30 * power(2, attempts)::int))),
    lease_until = NULL,
    last_error = sqlc.arg(last_error)::text
WHERE id = sqlc.arg(id)::uuid AND attempts = sqlc.arg(attempts)::int AND status = 'running';

-- name: CompleteAnswerEffect :execrows
UPDATE effect_outbox SET status = 'done', lease_until = NULL, last_error = ''
WHERE id = sqlc.arg(id)::uuid AND attempts = sqlc.arg(attempts)::int AND status = 'running';
