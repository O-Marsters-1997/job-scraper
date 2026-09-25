-- name: FindInterestedUsers :many
SELECT u.id AS user_id,
    COALESCE(sc.excluded_title_keywords, '{}')::text[] AS excluded_title_keywords,
    COALESCE(sc.excluded_companies, '{}')::text[] AS excluded_companies,
    COALESCE(sc.excluded_seniority, '{}')::text[] AS excluded_seniority,
    COALESCE(sc.excluded_locations, '{}')::text[] AS excluded_locations,
    COALESCE(sc.updated_at, 'epoch'::timestamptz) AS config_version,
    COALESCE(p.suitability_model, 'claude-haiku-4-5-20251001') AS model
FROM users u
LEFT JOIN search_config sc ON sc.user_id = u.id
LEFT JOIN user_ai_prefs p ON p.user_id = u.id
WHERE EXISTS (
    SELECT 1 FROM tracked_companies tc JOIN companies c ON c.id = tc.company_id
    WHERE tc.user_id = u.id AND tc.enabled AND
        (c.id = sqlc.narg(company_id)::uuid OR c.slug = sqlc.arg(company_slug)::text)
) OR EXISTS (
    SELECT 1 FROM source_targets st WHERE st.user_id = u.id AND st.enabled
        AND st.source = sqlc.arg(source)::text
        AND (sqlc.arg(discovery)::boolean OR st.value = sqlc.arg(company_slug)::text)
);

-- name: InsertScoringEffect :exec
INSERT INTO effect_outbox (job_id, user_id, fingerprint, config_version, model, first_discovery)
VALUES (sqlc.arg(job_id)::uuid, sqlc.arg(user_id)::uuid, sqlc.arg(fingerprint)::text,
    sqlc.arg(config_version)::timestamptz, sqlc.arg(model)::text, sqlc.arg(first_discovery)::boolean)
ON CONFLICT DO NOTHING;

-- name: QueueTrackingScores :exec
INSERT INTO effect_outbox (job_id, user_id, fingerprint, config_version, model)
SELECT j.id, sqlc.arg(user_id)::uuid, j.content_fingerprint,
    COALESCE(sc.updated_at, 'epoch'::timestamptz),
    COALESCE(p.suitability_model, 'claude-haiku-4-5-20251001')
FROM jobs j JOIN companies c ON c.id = sqlc.arg(company_id)::uuid
LEFT JOIN search_config sc ON sc.user_id = sqlc.arg(user_id)::uuid
LEFT JOIN user_ai_prefs p ON p.user_id = sqlc.arg(user_id)::uuid
WHERE j.closed_at IS NULL AND j.content_fingerprint IS NOT NULL
    AND (j.company_id = c.id OR j.company_slug = c.slug)
ON CONFLICT DO NOTHING;

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

-- name: QueueRescore :execrows
WITH interested AS (
    SELECT job_id FROM job_scores WHERE user_id = sqlc.arg(user_id)::uuid
    UNION
    SELECT job_id FROM effect_outbox WHERE user_id = sqlc.arg(user_id)::uuid
)
INSERT INTO effect_outbox (job_id, user_id, fingerprint, config_version, model)
SELECT j.id, sqlc.arg(user_id)::uuid, j.content_fingerprint,
    COALESCE(sc.updated_at, 'epoch'::timestamptz),
    COALESCE(p.suitability_model, 'claude-haiku-4-5-20251001')
FROM interested i JOIN jobs j ON j.id = i.job_id
LEFT JOIN search_config sc ON sc.user_id = sqlc.arg(user_id)::uuid
LEFT JOIN user_ai_prefs p ON p.user_id = sqlc.arg(user_id)::uuid
WHERE j.closed_at IS NULL
    AND j.content_fingerprint IS NOT NULL
    AND NOT EXISTS (SELECT 1 FROM effect_outbox e
        WHERE e.job_id = j.id AND e.user_id = sqlc.arg(user_id)::uuid
        AND e.fingerprint = j.content_fingerprint
        AND e.config_version = COALESCE(sc.updated_at, 'epoch'::timestamptz)
        AND e.model = COALESCE(p.suitability_model, 'claude-haiku-4-5-20251001'))
ORDER BY j.id LIMIT 100
ON CONFLICT DO NOTHING;

-- name: GetScoringStatus :one
SELECT
    (SELECT count(*) FROM effect_outbox WHERE user_id = sqlc.arg(user_id)::uuid AND status IN ('pending', 'running')) AS pending,
    (SELECT count(*) FROM effect_outbox WHERE user_id = sqlc.arg(user_id)::uuid AND status = 'failed') AS failed,
    (SELECT count(*) FROM job_scores s JOIN jobs j ON j.id = s.job_id
        LEFT JOIN search_config sc ON sc.user_id = s.user_id
        LEFT JOIN user_ai_prefs p ON p.user_id = s.user_id
        WHERE s.user_id = sqlc.arg(user_id)::uuid AND
            (s.score_fingerprint IS DISTINCT FROM j.content_fingerprint OR
             s.score_config_version IS DISTINCT FROM COALESCE(sc.updated_at, 'epoch'::timestamptz) OR
             s.score_model IS DISTINCT FROM COALESCE(p.suitability_model, 'claude-haiku-4-5-20251001'))) AS stale;

-- name: ClaimScoringEffect :one
WITH next AS (
    SELECT id FROM effect_outbox
    WHERE (status = 'pending' AND due_at <= NOW())
       OR (status = 'running' AND lease_until <= NOW())
    ORDER BY due_at, id LIMIT 1 FOR UPDATE SKIP LOCKED
)
UPDATE effect_outbox e SET status = 'running', attempts = attempts + 1,
    lease_until = NOW() + interval '5 minutes'
FROM next WHERE e.id = next.id
RETURNING e.id, e.job_id, e.user_id, e.fingerprint, e.config_version, e.model, e.attempts, e.first_discovery;

-- name: FailScoringEffect :exec
UPDATE effect_outbox SET
    status = CASE WHEN sqlc.arg(terminal)::bool OR attempts >= 8 THEN 'failed' ELSE 'pending' END,
    due_at = NOW() + make_interval(secs =>
        COALESCE(sqlc.narg(retry_after_secs)::int, LEAST(3600, 30 * power(2, attempts)::int))),
    lease_until = NULL,
    last_error = sqlc.arg(last_error)::text
WHERE id = sqlc.arg(id)::uuid AND attempts = sqlc.arg(attempts)::int AND status = 'running';

-- name: CompleteScoringEffect :execrows
WITH completed AS (
    UPDATE effect_outbox SET status = 'done', lease_until = NULL, last_error = ''
    WHERE id = sqlc.arg(id)::uuid AND attempts = sqlc.arg(attempts)::int AND status = 'running'
    RETURNING job_id, user_id, fingerprint, config_version, model
)
INSERT INTO job_scores (job_id, user_id, suitability_score, reasoning, matched, missing,
    suitability_skipped, score_fingerprint, score_config_version, score_model)
SELECT e.job_id, e.user_id, sqlc.arg(score)::int, sqlc.arg(reasoning)::text,
    sqlc.arg(matched)::text[], sqlc.arg(missing)::text[], false,
    e.fingerprint, e.config_version, e.model
FROM completed e JOIN jobs j ON j.id = e.job_id
LEFT JOIN search_config sc ON sc.user_id = e.user_id
LEFT JOIN user_ai_prefs p ON p.user_id = e.user_id
WHERE j.content_fingerprint = e.fingerprint
    AND COALESCE(sc.updated_at, 'epoch'::timestamptz) = e.config_version
    AND COALESCE(p.suitability_model, 'claude-haiku-4-5-20251001') = e.model
ON CONFLICT (job_id, user_id) DO UPDATE SET
    suitability_score = EXCLUDED.suitability_score, reasoning = EXCLUDED.reasoning,
    matched = EXCLUDED.matched, missing = EXCLUDED.missing, suitability_skipped = false,
    score_fingerprint = EXCLUDED.score_fingerprint,
    score_config_version = EXCLUDED.score_config_version, score_model = EXCLUDED.score_model,
    updated_at = NOW();
