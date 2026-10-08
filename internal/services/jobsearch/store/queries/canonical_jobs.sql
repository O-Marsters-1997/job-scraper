-- name: LockCanonicalJob :exec
SELECT pg_advisory_xact_lock(hashtextextended($1, 0));

-- name: FindCanonicalJob :one
SELECT j.id, j.url, COALESCE(j.content_fingerprint, '') AS content_fingerprint,
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
    content_fingerprint, content_changed_at, match_title, match_location)
VALUES (sqlc.arg(title), sqlc.arg(location), sqlc.arg(url), sqlc.arg(company_slug),
    sqlc.arg(source), sqlc.arg(updated_at), sqlc.arg(description), sqlc.arg(salary_raw),
    sqlc.arg(work_arrangement), COALESCE(sqlc.narg(company_id)::uuid, (SELECT id FROM companies WHERE slug = sqlc.arg(company_slug))),
    sqlc.narg(board_id)::uuid, sqlc.narg(posting_id),
    sqlc.arg(fingerprint), NOW(), sqlc.arg(match_title), sqlc.arg(match_location))
RETURNING id;

-- name: UpdateChangedCanonicalJob :exec
UPDATE jobs SET title = sqlc.arg(title), location = sqlc.arg(location),
    updated_at = sqlc.arg(updated_at), description = sqlc.arg(description),
    salary_raw = sqlc.arg(salary_raw), work_arrangement = sqlc.arg(work_arrangement),
    content_fingerprint = sqlc.arg(fingerprint), content_changed_at = NOW(),
    match_title = sqlc.arg(match_title), match_location = sqlc.arg(match_location),
    company_id = COALESCE(sqlc.narg(company_id)::uuid, company_id, (SELECT id FROM companies WHERE slug = sqlc.arg(company_slug))),
    primary_board_id = COALESCE(sqlc.narg(board_id)::uuid, primary_board_id),
    provider_posting_id = COALESCE(sqlc.narg(posting_id), provider_posting_id),
    scraped_at = NOW()
WHERE id = sqlc.arg(id)::uuid;

-- name: UpgradeCanonicalJob :exec
UPDATE jobs SET title = sqlc.arg(title), location = sqlc.arg(location), url = sqlc.arg(url),
    source = sqlc.arg(source), updated_at = sqlc.arg(updated_at), description = sqlc.arg(description),
    salary_raw = sqlc.arg(salary_raw), work_arrangement = sqlc.arg(work_arrangement),
    content_fingerprint = sqlc.arg(fingerprint), content_changed_at = NOW(),
    match_title = sqlc.arg(match_title), match_location = sqlc.arg(match_location),
    company_id = COALESCE(sqlc.narg(company_id)::uuid, company_id, (SELECT id FROM companies WHERE slug = sqlc.arg(company_slug))),
    primary_board_id = sqlc.arg(board_id)::uuid, provider_posting_id = sqlc.arg(posting_id),
    scraped_at = NOW()
WHERE id = sqlc.arg(id)::uuid;

-- name: FindMatchCandidates :many
SELECT id, url, COALESCE(match_location, '') AS match_location, primary_board_id
FROM jobs
WHERE match_title = sqlc.arg(match_title)
    AND (company_slug = sqlc.arg(company_slug) OR company_id = sqlc.narg(company_id)::uuid)
    AND closed_at IS NULL
    AND updated_at > NOW() - INTERVAL '90 days'
FOR UPDATE;

-- name: UpdateUnchangedCanonicalJob :exec
UPDATE jobs SET company_id = COALESCE(sqlc.narg(company_id)::uuid, company_id, (SELECT id FROM companies WHERE slug = sqlc.arg(company_slug))),
    primary_board_id = COALESCE(sqlc.narg(board_id)::uuid, primary_board_id),
    provider_posting_id = COALESCE(sqlc.narg(posting_id), provider_posting_id),
    content_fingerprint = COALESCE(content_fingerprint, sqlc.arg(fingerprint))
WHERE id = sqlc.arg(id)::uuid;

-- name: SaveCanonicalJobAlias :execrows
INSERT INTO job_urls (job_id, normalized_url, source)
VALUES ($1::uuid, $2, $3)
ON CONFLICT (normalized_url) DO UPDATE SET last_seen_at = NOW()
WHERE job_urls.job_id = EXCLUDED.job_id;

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
