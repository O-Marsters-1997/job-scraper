-- name: UpsertCompany :one
INSERT INTO companies (slug, name, ats_source, ats_token, domain, linkedin_company_id)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (slug) DO UPDATE SET
    ats_source          = COALESCE(companies.ats_source, EXCLUDED.ats_source),
    ats_token           = COALESCE(companies.ats_token,  EXCLUDED.ats_token),
    domain              = COALESCE(companies.domain, EXCLUDED.domain),
    linkedin_company_id = COALESCE(companies.linkedin_company_id, EXCLUDED.linkedin_company_id),
    updated_at = NOW()
RETURNING *;

-- name: GetCompany :one
SELECT * FROM companies WHERE id = $1;

-- name: PageCompaniesForUser :many
SELECT c.id, c.slug, c.name, c.ats_source, c.ats_token, c.domain, c.linkedin_company_id,
    c.last_crawled_at, c.first_seen_at,
    (SELECT COUNT(*) FROM jobs j JOIN job_scores js ON js.job_id = j.id AND js.user_id = sqlc.arg(user_id)::uuid
     WHERE j.company_id = c.id AND j.closed_at IS NULL
       AND NOT js.breakdown @> '[{"effect":"blocked"}]'::jsonb) AS job_count,
    COALESCE(tc.enabled, FALSE) AS tracked,
    COALESCE(tc.review_state, '')::text AS review_state,
    tc.check_interval_minutes,
    (SELECT MAX(bps.last_completed_at)::timestamptz FROM company_boards cb
     JOIN board_poll_state bps ON bps.board_id = cb.id
     WHERE cb.company_id = c.id AND cb.status = 'verified') AS last_checked_at
FROM companies c
LEFT JOIN tracked_companies tc ON tc.user_id = sqlc.arg(user_id)::uuid AND tc.company_id = c.id
WHERE (sqlc.narg(cursor_id)::uuid IS NULL OR (c.name, c.id) > (sqlc.narg(cursor_name)::text, sqlc.narg(cursor_id)::uuid))
  AND (sqlc.arg(search)::text = ''
       OR strpos(lower(c.name), lower(sqlc.arg(search)::text)) > 0
       OR strpos(c.slug, lower(sqlc.arg(search)::text)) > 0)
  AND (NOT sqlc.arg(tracked_only)::bool OR COALESCE(tc.enabled, FALSE))
ORDER BY c.name, c.id
LIMIT sqlc.arg(page_limit)::int;

-- name: GetCompanyForUser :one
SELECT c.id, c.slug, c.name, c.ats_source, c.ats_token, c.domain, c.linkedin_company_id,
    c.last_crawled_at, c.first_seen_at,
    (SELECT COUNT(*) FROM jobs j JOIN job_scores js ON js.job_id = j.id AND js.user_id = $1
     WHERE j.company_id = c.id AND j.closed_at IS NULL
       AND NOT js.breakdown @> '[{"effect":"blocked"}]'::jsonb) AS job_count,
    COALESCE(tc.enabled, FALSE) AS tracked,
    COALESCE(tc.review_state, '')::text AS review_state,
    tc.check_interval_minutes,
    (SELECT MAX(bps.last_completed_at)::timestamptz FROM company_boards cb
     JOIN board_poll_state bps ON bps.board_id = cb.id
     WHERE cb.company_id = c.id AND cb.status = 'verified') AS last_checked_at
FROM companies c
LEFT JOIN tracked_companies tc ON tc.user_id = $1 AND tc.company_id = c.id
WHERE c.id = $2;

-- name: SetCompanyTracking :one
INSERT INTO tracked_companies (user_id, company_id, enabled, check_interval_minutes)
VALUES ($1, $2, $3, COALESCE(NULLIF(sqlc.arg(check_interval_minutes)::int, 0), 360))
ON CONFLICT (user_id, company_id) DO UPDATE SET
    enabled = EXCLUDED.enabled,
    check_interval_minutes = COALESCE(NULLIF(sqlc.arg(check_interval_minutes)::int, 0), tracked_companies.check_interval_minutes),
    updated_at = NOW()
RETURNING user_id, company_id, enabled, review_state, check_interval_minutes;

-- name: TrackDiscoveredCompany :execrows
INSERT INTO tracked_companies (user_id, company_id, review_state)
VALUES ($1, $2, 'new')
ON CONFLICT (user_id, company_id) DO NOTHING;

-- name: SetCompanyReviewState :one
UPDATE tracked_companies
SET review_state = sqlc.arg(review_state)::text,
    enabled = sqlc.arg(review_state)::text <> 'dismissed',
    updated_at = NOW()
WHERE user_id = $1 AND company_id = $2
RETURNING user_id, company_id, enabled, review_state, check_interval_minutes;

-- name: ListCompaniesToCrawl :many
SELECT * FROM companies
WHERE domain IS NOT NULL AND ats_source IS NULL
  AND (last_crawled_at IS NULL OR last_crawled_at < NOW() - make_interval(days => 30))
ORDER BY last_crawled_at NULLS FIRST LIMIT $1;

-- name: TouchCompanyCrawled :exec
UPDATE companies SET last_crawled_at = NOW(), updated_at = NOW() WHERE id = $1;

-- name: ListTrackedCompaniesForUser :many
SELECT c.id, c.name, c.slug, tc.enabled, tc.review_state, tc.check_interval_minutes,
    (SELECT COUNT(*) FROM jobs j WHERE j.company_id = c.id AND j.closed_at IS NULL) AS open_jobs,
    (SELECT COUNT(*) FROM jobs j JOIN job_scores js ON js.job_id = j.id AND js.user_id = tc.user_id
     WHERE j.company_id = c.id AND j.closed_at IS NULL
       AND NOT js.breakdown @> '[{"effect":"blocked"}]'::jsonb) AS relevant_jobs,
    (SELECT MAX(bps.last_completed_at)::timestamptz FROM company_boards cb
     JOIN board_poll_state bps ON bps.board_id = cb.id
     WHERE cb.company_id = c.id AND cb.status = 'verified') AS last_checked_at
FROM tracked_companies tc
JOIN companies c ON c.id = tc.company_id
WHERE tc.user_id = $1
ORDER BY c.name, c.id;

-- name: DeleteCompanyTracking :execrows
DELETE FROM tracked_companies WHERE user_id = $1 AND company_id = $2;

-- name: ListNewCompaniesForUser :many
SELECT c.id, c.name, c.slug, tc.created_at AS added_at
FROM tracked_companies tc
JOIN companies c ON c.id = tc.company_id
WHERE tc.user_id = $1 AND tc.review_state = 'new'
ORDER BY tc.created_at DESC, c.id;

-- name: ListNewCompanyJobs :many
SELECT j.company_id, j.company_slug, j.title, j.location, js.suitability_score
FROM tracked_companies tc
JOIN jobs j ON j.company_id = tc.company_id AND j.closed_at IS NULL
LEFT JOIN job_scores js ON js.job_id = j.id AND js.user_id = tc.user_id
WHERE tc.user_id = $1 AND tc.review_state = 'new';

-- name: RenameCompany :exec
UPDATE companies SET name = $2, updated_at = NOW() WHERE id = $1;
