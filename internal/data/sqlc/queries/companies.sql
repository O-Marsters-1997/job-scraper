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

-- name: ListCompaniesToCrawl :many
SELECT * FROM companies
WHERE domain IS NOT NULL AND ats_source IS NULL
  AND (last_crawled_at IS NULL OR last_crawled_at < NOW() - make_interval(days => 30))
ORDER BY last_crawled_at NULLS FIRST LIMIT $1;

-- name: TouchCompanyCrawled :exec
UPDATE companies SET last_crawled_at = NOW(), updated_at = NOW() WHERE id = $1;

-- name: ListCompaniesForUser :many
SELECT c.id, c.slug, c.name, c.ats_source, c.ats_token, c.domain, c.linkedin_company_id,
    c.last_crawled_at, c.first_seen_at,
    (SELECT COUNT(*) FROM jobs j WHERE j.company_slug = c.slug) AS job_count,
    st.id AS target_id,
    COALESCE(st.enabled, FALSE) AS tracked,
    st.check_interval_minutes,
    st.last_checked_at
FROM companies c
LEFT JOIN source_targets st
    ON st.user_id = $1 AND st.source = c.ats_source AND st.value = c.ats_token
ORDER BY c.name;
