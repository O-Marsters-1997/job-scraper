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

-- name: ListCompaniesForUser :many
SELECT c.id, c.slug, c.name, c.ats_source, c.ats_token, c.domain, c.linkedin_company_id,
    c.last_crawled_at, c.first_seen_at,
    (SELECT COUNT(*) FROM jobs j WHERE j.company_slug = c.slug) AS job_count,
    COALESCE(tc.enabled, FALSE) AS tracked,
    COALESCE(tc.review_state, '')::text AS review_state,
    tc.check_interval_minutes,
    (SELECT MAX(bps.last_completed_at)::timestamptz FROM company_boards cb
     JOIN board_poll_state bps ON bps.board_id = cb.id
     WHERE cb.company_id = c.id AND cb.status = 'verified') AS last_checked_at
FROM companies c
LEFT JOIN tracked_companies tc ON tc.user_id = $1 AND tc.company_id = c.id
ORDER BY c.name;

-- name: SetCompanyTracking :one
INSERT INTO tracked_companies (user_id, company_id, enabled, check_interval_minutes)
VALUES ($1, $2, $3, COALESCE(NULLIF(sqlc.arg(check_interval_minutes)::int, 0), 360))
ON CONFLICT (user_id, company_id) DO UPDATE SET
    enabled = EXCLUDED.enabled,
    check_interval_minutes = COALESCE(NULLIF(sqlc.arg(check_interval_minutes)::int, 0), tracked_companies.check_interval_minutes),
    updated_at = NOW()
RETURNING user_id, company_id, enabled, review_state, check_interval_minutes;

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
