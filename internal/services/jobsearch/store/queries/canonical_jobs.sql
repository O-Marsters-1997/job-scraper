-- name: LockCanonicalJob :exec
SELECT pg_advisory_xact_lock(hashtextextended($1, 0));

-- name: FindCanonicalJob :one
SELECT j.id::text AS id, j.url, COALESCE(j.content_fingerprint, '') AS content_fingerprint,
    j.title, j.description, j.location, j.salary_raw, j.work_arrangement,
    j.primary_board_id, j.provider_posting_id
FROM jobs j
LEFT JOIN job_urls u ON u.job_id = j.id AND u.normalized_url = sqlc.arg(url)
WHERE (j.primary_board_id = sqlc.narg(board_id)::uuid AND j.provider_posting_id = sqlc.narg(posting_id))
    OR u.normalized_url IS NOT NULL OR j.url = sqlc.arg(url)
ORDER BY CASE WHEN j.primary_board_id = sqlc.narg(board_id)::uuid
    AND j.provider_posting_id = sqlc.narg(posting_id) THEN 0
    WHEN u.normalized_url IS NOT NULL THEN 1 ELSE 2 END
LIMIT 1 FOR UPDATE OF j;

-- name: InsertCanonicalJob :one
INSERT INTO jobs (title, location, url, company_slug, source, updated_at, description,
    salary_raw, work_arrangement, company_id, primary_board_id, provider_posting_id,
    content_fingerprint, content_changed_at)
VALUES (sqlc.arg(title), sqlc.arg(location), sqlc.arg(url), sqlc.arg(company_slug),
    sqlc.arg(source), sqlc.arg(updated_at), sqlc.arg(description), sqlc.arg(salary_raw),
    sqlc.arg(work_arrangement), sqlc.narg(company_id)::uuid,
    sqlc.narg(board_id)::uuid, sqlc.narg(posting_id),
    sqlc.arg(fingerprint), NOW())
RETURNING id::text;

-- name: UpdateChangedCanonicalJob :exec
UPDATE jobs SET title = sqlc.arg(title), location = sqlc.arg(location),
    updated_at = sqlc.arg(updated_at), description = sqlc.arg(description),
    salary_raw = sqlc.arg(salary_raw), work_arrangement = sqlc.arg(work_arrangement),
    content_fingerprint = sqlc.arg(fingerprint), content_changed_at = NOW(),
    company_id = COALESCE(sqlc.narg(company_id)::uuid, company_id),
    primary_board_id = COALESCE(sqlc.narg(board_id)::uuid, primary_board_id),
    provider_posting_id = COALESCE(sqlc.narg(posting_id), provider_posting_id),
    scraped_at = NOW()
WHERE id = sqlc.arg(id)::uuid;

-- name: UpdateUnchangedCanonicalJob :exec
UPDATE jobs SET company_id = COALESCE(sqlc.narg(company_id)::uuid, company_id),
    primary_board_id = COALESCE(sqlc.narg(board_id)::uuid, primary_board_id),
    provider_posting_id = COALESCE(sqlc.narg(posting_id), provider_posting_id),
    content_fingerprint = COALESCE(content_fingerprint, sqlc.arg(fingerprint))
WHERE id = sqlc.arg(id)::uuid;

-- name: SaveCanonicalJobAlias :execrows
INSERT INTO job_urls (job_id, normalized_url, source)
VALUES ($1::uuid, $2, $3)
ON CONFLICT (normalized_url) DO UPDATE SET last_seen_at = NOW()
WHERE job_urls.job_id = EXCLUDED.job_id;

-- name: DeleteStaleOptionAnswers :exec
-- Writes scoring's option_answers table directly; documented exception
-- (ADR 0011, issue #264) pending #267.
DELETE FROM option_answers WHERE job_id = sqlc.arg(job_id)::uuid AND fingerprint != sqlc.arg(fingerprint)::text;
