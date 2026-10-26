-- name: UpsertCompany :one
INSERT INTO companies (slug, name, ats_source, ats_token, domain, linkedin_company_id, kind)
VALUES ($1, $2, $3, $4, $5, $6, COALESCE(NULLIF(sqlc.arg(kind)::text, ''), 'unclassified'))
ON CONFLICT (slug) DO UPDATE SET
    ats_source          = COALESCE(companies.ats_source, EXCLUDED.ats_source),
    ats_token           = COALESCE(companies.ats_token,  EXCLUDED.ats_token),
    domain              = COALESCE(companies.domain, EXCLUDED.domain),
    linkedin_company_id = COALESCE(companies.linkedin_company_id, EXCLUDED.linkedin_company_id),
    kind                = CASE
        WHEN companies.kind = 'agency' OR sqlc.arg(kind)::text = '' THEN companies.kind
        ELSE sqlc.arg(kind)::text
    END,
    updated_at = NOW()
RETURNING *;

-- name: GetCompany :one
SELECT * FROM companies WHERE id = $1;

-- name: PageCompaniesForUser :many
SELECT COUNT(*) OVER ()::bigint AS total, c.id, c.slug, c.name,
    COALESCE((SELECT cb.source FROM company_boards cb
     WHERE cb.company_id = c.id AND cb.status = 'verified'
     ORDER BY cb.source = 'wttj', cb.verified_at, cb.id LIMIT 1), '')::text AS ats_source,
    c.ats_token, c.domain, c.linkedin_company_id,
    c.last_crawled_at, c.first_seen_at, c.kind,
    COALESCE(jc.n, 0)::bigint AS job_count,
    COALESCE(tc.enabled, FALSE) AS tracked,
    EXISTS (SELECT 1 FROM company_favourites cf WHERE cf.user_id = sqlc.arg(user_id)::uuid AND cf.company_id = c.id) AS favourite,
    COALESCE(tc.review_state, '')::text AS review_state,
    tc.check_interval_minutes,
    (SELECT MAX(bps.last_completed_at)::timestamptz FROM company_boards cb
     JOIN board_poll_state bps ON bps.board_id = cb.id
     WHERE cb.company_id = c.id AND cb.status = 'verified') AS last_checked_at
FROM companies c
LEFT JOIN tracked_companies tc ON tc.user_id = sqlc.arg(user_id)::uuid AND tc.company_id = c.id
LEFT JOIN (
    SELECT j.company_id, COUNT(*) AS n FROM job_scores js JOIN jobs j ON j.id = js.job_id
    WHERE js.user_id = sqlc.arg(user_id)::uuid AND j.closed_at IS NULL
      AND NOT js.breakdown @> '[{"effect":"blocked"}]'::jsonb
    GROUP BY j.company_id
) jc ON jc.company_id = c.id
WHERE (sqlc.arg(search)::text = ''
       OR strpos(lower(c.name), lower(sqlc.arg(search)::text)) > 0
       OR strpos(c.slug, lower(sqlc.arg(search)::text)) > 0)
  AND (NOT sqlc.arg(tracked_only)::bool OR COALESCE(tc.enabled, FALSE))
  AND (NOT sqlc.arg(favourite_only)::bool OR EXISTS (SELECT 1 FROM company_favourites cf WHERE cf.user_id = sqlc.arg(user_id)::uuid AND cf.company_id = c.id))
  AND (NOT sqlc.arg(no_board_only)::bool OR NOT EXISTS (SELECT 1 FROM company_boards cb WHERE cb.company_id = c.id))
  AND (sqlc.arg(poster)::text = ''
       OR (sqlc.arg(poster)::text = 'agency') = (c.kind IN ('agency', 'likely_agency')))
  AND (COALESCE(cardinality(sqlc.arg(ats_sources)::text[]), 0) = 0
       OR EXISTS (SELECT 1 FROM company_boards cb
                  WHERE cb.company_id = c.id AND cb.status = 'verified'
                    AND cb.source = ANY(sqlc.arg(ats_sources)::text[])))
ORDER BY
    CASE WHEN sqlc.arg(sort)::text = 'relevance'
          AND EXISTS (SELECT 1 FROM company_favourites cf WHERE cf.user_id = sqlc.arg(user_id)::uuid AND cf.company_id = c.id)
         THEN 0 ELSE 1 END,
    CASE WHEN sqlc.arg(sort)::text = 'relevance' AND COALESCE(tc.enabled, FALSE) THEN 0 ELSE 1 END,
    CASE WHEN sqlc.arg(sort)::text IN ('relevance', 'open_jobs') THEN COALESCE(jc.n, 0) ELSE 0 END DESC,
    c.name, c.id
LIMIT sqlc.arg(page_limit)::int OFFSET sqlc.arg(page_offset)::int;

-- name: CountCompaniesByATS :many
SELECT source, COUNT(DISTINCT company_id)::bigint AS companies
FROM company_boards
WHERE status = 'verified' AND source <> 'wttj'
GROUP BY source
ORDER BY source;

-- name: CountCompanyBoardCoverage :one
SELECT COUNT(*)::bigint AS all_companies,
    COUNT(*) FILTER (WHERE EXISTS (SELECT 1 FROM company_boards cb
        WHERE cb.company_id = c.id AND cb.status = 'verified' AND cb.source <> 'wttj'))::bigint AS verified_companies
FROM companies c;

-- name: GetCompanyForUser :one
SELECT c.id, c.slug, c.name,
    COALESCE((SELECT cb.source FROM company_boards cb
     WHERE cb.company_id = c.id AND cb.status = 'verified'
     ORDER BY cb.source = 'wttj', cb.verified_at, cb.id LIMIT 1), '')::text AS ats_source,
    c.ats_token, c.domain, c.linkedin_company_id,
    c.last_crawled_at, c.first_seen_at, c.kind,
    (SELECT COUNT(*) FROM jobs j JOIN job_scores js ON js.job_id = j.id AND js.user_id = $1
     WHERE j.company_id = c.id AND j.closed_at IS NULL
       AND NOT js.breakdown @> '[{"effect":"blocked"}]'::jsonb) AS job_count,
    COALESCE(tc.enabled, FALSE) AS tracked,
    EXISTS (SELECT 1 FROM company_favourites cf WHERE cf.user_id = $1 AND cf.company_id = c.id) AS favourite,
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
    COALESCE(jc.open_jobs, 0) AS open_jobs,
    COALESCE(jc.relevant_jobs, 0) AS relevant_jobs,
    (SELECT MAX(bps.last_completed_at)::timestamptz FROM company_boards cb
     JOIN board_poll_state bps ON bps.board_id = cb.id
     WHERE cb.company_id = c.id AND cb.status = 'verified') AS last_checked_at
FROM tracked_companies tc
JOIN companies c ON c.id = tc.company_id
LEFT JOIN (
    SELECT j.company_id, COUNT(*) AS open_jobs,
        COUNT(*) FILTER (WHERE js.job_id IS NOT NULL
                           AND NOT js.breakdown @> '[{"effect":"blocked"}]'::jsonb) AS relevant_jobs
    FROM tracked_companies t
    JOIN jobs j ON j.company_id = t.company_id AND j.closed_at IS NULL
    LEFT JOIN job_scores js ON js.job_id = j.id AND js.user_id = t.user_id
    WHERE t.user_id = $1
    GROUP BY j.company_id
) jc ON jc.company_id = c.id
WHERE tc.user_id = $1
ORDER BY c.name, c.id;

-- name: DeleteCompanyTracking :execrows
DELETE FROM tracked_companies WHERE user_id = $1 AND company_id = $2;

-- name: ListNewCompaniesForUser :many
SELECT c.id, c.name, c.slug, tc.created_at AS added_at
FROM tracked_companies tc
JOIN companies c ON c.id = tc.company_id
WHERE tc.user_id = $1 AND tc.review_state = 'new'
ORDER BY tc.created_at DESC, c.id
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);

-- name: CountNewCompaniesForUser :one
SELECT COUNT(*) FROM tracked_companies WHERE user_id = $1 AND review_state = 'new';

-- name: ListNewCompanyJobs :many
SELECT j.company_id, j.company_slug, j.title, j.location, js.suitability_score
FROM tracked_companies tc
JOIN jobs j ON j.company_id = tc.company_id AND j.closed_at IS NULL
LEFT JOIN job_scores js ON js.job_id = j.id AND js.user_id = tc.user_id
WHERE tc.user_id = $1 AND tc.review_state = 'new' AND tc.company_id = ANY(sqlc.arg(company_ids)::uuid[]);

-- name: RenameCompany :exec
UPDATE companies SET name = $2, updated_at = NOW() WHERE id = $1;

-- name: AddCompanyFavourite :exec
INSERT INTO company_favourites (user_id, company_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: RemoveCompanyFavourite :exec
DELETE FROM company_favourites WHERE user_id = $1 AND company_id = $2;

-- name: KeepNewTrackedCompany :exec
UPDATE tracked_companies
SET review_state = 'kept', updated_at = NOW()
WHERE user_id = $1 AND company_id = $2 AND review_state <> 'kept';

-- name: ResetTrackedCompanyInterval :exec
UPDATE tracked_companies
SET check_interval_minutes = 360, updated_at = NOW()
WHERE user_id = $1 AND company_id = $2;

-- name: SetCompanyKind :execrows
UPDATE companies SET kind = $2, updated_at = NOW() WHERE id = $1;

-- name: SuggestCompanyKind :execrows
UPDATE companies SET kind = $2, updated_at = NOW() WHERE id = $1 AND kind = 'unclassified';
