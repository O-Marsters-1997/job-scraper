-- name: QueueAnswerEffect :exec
-- Writes scoring's effect_outbox table directly; documented exception
-- (ADR 0011, issue #264) pending #267.
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
-- Writes scoring's effect_outbox table directly; documented exception
-- (ADR 0011, issue #264) pending #267.
INSERT INTO effect_outbox (job_id, fingerprint)
SELECT j.id, j.content_fingerprint
FROM jobs j JOIN companies c ON c.id = sqlc.arg(company_id)::uuid
WHERE j.closed_at IS NULL AND j.content_fingerprint IS NOT NULL
    AND (j.company_id = c.id OR j.company_slug = c.slug)
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
