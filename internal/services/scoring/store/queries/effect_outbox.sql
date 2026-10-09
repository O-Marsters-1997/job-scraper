-- name: QueueOptionBackfill :exec
INSERT INTO effect_outbox (job_id, fingerprint)
SELECT DISTINCT j.id, j.content_fingerprint
FROM jobs j JOIN job_scores s ON s.job_id = j.id
WHERE j.closed_at IS NULL AND j.content_fingerprint IS NOT NULL
ON CONFLICT (job_id, fingerprint, model) WHERE status IN ('pending', 'running') DO NOTHING;

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

-- name: QueueMissingAnswers :execrows
INSERT INTO effect_outbox (job_id, fingerprint, model, first_discovery)
SELECT DISTINCT j.id, j.content_fingerprint, sqlc.arg(model)::text, FALSE
FROM job_scores s
JOIN jobs j ON j.id = s.job_id
WHERE s.user_id = sqlc.arg(user_id)::uuid
    AND j.closed_at IS NULL
    AND j.content_fingerprint IS NOT NULL
    AND EXISTS (
        SELECT 1 FROM unnest(sqlc.arg(question_hashes)::text[]) AS missing(hash)
        WHERE NOT EXISTS (
            SELECT 1 FROM option_answers a
            WHERE a.user_id = s.user_id AND a.job_id = j.id AND a.fingerprint = j.content_fingerprint
                AND a.model = sqlc.arg(model)::text AND a.question_hash = missing.hash
        )
    )
ON CONFLICT (job_id, fingerprint, model) WHERE status IN ('pending', 'running') DO NOTHING;

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
