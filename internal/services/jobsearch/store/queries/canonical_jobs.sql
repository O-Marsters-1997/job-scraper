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
INSERT INTO jobs (title, location, url, company_slug, source, posted_at, posted_after, description,
    salary_raw, salary_min, salary_max, work_arrangement, company_id, primary_board_id, provider_posting_id,
    content_fingerprint, content_changed_at, match_title, match_location)
VALUES (sqlc.arg(title), sqlc.arg(location), sqlc.arg(url), sqlc.arg(company_slug),
    sqlc.arg(source), sqlc.narg(posted_at)::timestamptz,
    (SELECT last_completed_at FROM board_poll_state WHERE board_id = sqlc.narg(board_id)::uuid),
    sqlc.arg(description), sqlc.arg(salary_raw),
    sqlc.narg(salary_min)::float8, sqlc.narg(salary_max)::float8,
    sqlc.arg(work_arrangement), COALESCE(sqlc.narg(company_id)::uuid, (SELECT id FROM companies WHERE slug = sqlc.arg(company_slug))),
    sqlc.narg(board_id)::uuid, sqlc.narg(posting_id),
    sqlc.arg(fingerprint), NOW(), sqlc.arg(match_title), sqlc.arg(match_location))
RETURNING id;

-- name: UpdateChangedCanonicalJob :exec
UPDATE jobs SET title = sqlc.arg(title), location = sqlc.arg(location),
    reposted_at = CASE WHEN sqlc.narg(posted_at)::timestamptz > posted_at + INTERVAL '2 days'
        THEN GREATEST(reposted_at, sqlc.narg(posted_at)::timestamptz) ELSE reposted_at END,
    posted_at = LEAST(posted_at, sqlc.narg(posted_at)::timestamptz), description = sqlc.arg(description),
    salary_raw = sqlc.arg(salary_raw), salary_min = sqlc.narg(salary_min)::float8,
    salary_max = sqlc.narg(salary_max)::float8, work_arrangement = sqlc.arg(work_arrangement),
    content_fingerprint = sqlc.arg(fingerprint), content_changed_at = NOW(),
    match_title = sqlc.arg(match_title), match_location = sqlc.arg(match_location),
    company_id = COALESCE(sqlc.narg(company_id)::uuid, company_id, (SELECT id FROM companies WHERE slug = sqlc.arg(company_slug))),
    primary_board_id = COALESCE(sqlc.narg(board_id)::uuid, primary_board_id),
    provider_posting_id = COALESCE(sqlc.narg(posting_id), provider_posting_id),
    scraped_at = NOW()
WHERE id = sqlc.arg(id)::uuid;

-- name: UpgradeCanonicalJob :exec
UPDATE jobs SET title = sqlc.arg(title), location = sqlc.arg(location), url = sqlc.arg(url),
    source = sqlc.arg(source),
    reposted_at = CASE WHEN sqlc.narg(posted_at)::timestamptz > posted_at + INTERVAL '2 days'
        THEN GREATEST(reposted_at, sqlc.narg(posted_at)::timestamptz) ELSE reposted_at END,
    posted_at = LEAST(posted_at, sqlc.narg(posted_at)::timestamptz), description = sqlc.arg(description),
    salary_raw = sqlc.arg(salary_raw), salary_min = sqlc.narg(salary_min)::float8,
    salary_max = sqlc.narg(salary_max)::float8, work_arrangement = sqlc.arg(work_arrangement),
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
    AND COALESCE(posted_at, posted_after, first_discovered_at) > NOW() - INTERVAL '90 days'
FOR UPDATE;

-- name: UpdateUnchangedCanonicalJob :exec
UPDATE jobs SET company_id = COALESCE(sqlc.narg(company_id)::uuid, company_id, (SELECT id FROM companies WHERE slug = sqlc.arg(company_slug))),
    primary_board_id = COALESCE(sqlc.narg(board_id)::uuid, primary_board_id),
    provider_posting_id = COALESCE(sqlc.narg(posting_id), provider_posting_id),
    content_fingerprint = COALESCE(content_fingerprint, sqlc.arg(fingerprint)),
    reposted_at = CASE WHEN sqlc.narg(posted_at)::timestamptz > posted_at + INTERVAL '2 days'
        THEN GREATEST(reposted_at, sqlc.narg(posted_at)::timestamptz) ELSE reposted_at END,
    posted_at = LEAST(posted_at, sqlc.narg(posted_at)::timestamptz)
WHERE id = sqlc.arg(id)::uuid;

-- name: ApplyPostedDate :exec
UPDATE jobs SET
    reposted_at = CASE WHEN sqlc.arg(posted_at)::timestamptz > posted_at + INTERVAL '2 days'
        THEN GREATEST(reposted_at, sqlc.arg(posted_at)::timestamptz) ELSE reposted_at END,
    posted_at = LEAST(posted_at, sqlc.arg(posted_at)::timestamptz)
WHERE id = sqlc.arg(id)::uuid;

-- name: FindJobIDByURL :one
SELECT id FROM jobs WHERE url = sqlc.arg(url)
UNION ALL
SELECT job_id FROM job_urls WHERE normalized_url = sqlc.arg(url)
LIMIT 1;

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

-- name: FindJobByPosting :one
SELECT j.id, j.title, j.location, j.url, j.company_slug, j.source, j.posted_at, j.posted_after,
    j.reposted_at, j.scraped_at, j.first_discovered_at, j.description, j.salary_raw,
    j.work_arrangement, j.company_id, j.primary_board_id, j.provider_posting_id, j.content_fingerprint
FROM jobs j
WHERE (sqlc.arg(posting_id)::text <> '' AND j.provider_posting_id = sqlc.arg(posting_id)::text
        AND j.primary_board_id = (SELECT b.id FROM company_boards b
            WHERE b.source = sqlc.arg(source)::text AND b.board_token = sqlc.arg(token)::text))
    OR j.url = sqlc.arg(url)::text
    OR EXISTS (SELECT 1 FROM job_urls u WHERE u.job_id = j.id AND u.normalized_url = sqlc.arg(url)::text)
ORDER BY j.first_discovered_at
LIMIT 1;
