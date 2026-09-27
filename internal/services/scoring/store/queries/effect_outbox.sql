-- name: QueueOptionBackfill :exec
INSERT INTO effect_outbox (job_id, fingerprint)
SELECT DISTINCT j.id, j.content_fingerprint
FROM jobs j JOIN job_scores s ON s.job_id = j.id
WHERE j.closed_at IS NULL AND j.content_fingerprint IS NOT NULL
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
