-- name: UpsertCompany :one
INSERT INTO companies (slug, name, ats_source, ats_token)
VALUES ($1, $2, $3, $4)
ON CONFLICT (slug) DO UPDATE SET
    ats_source = COALESCE(companies.ats_source, EXCLUDED.ats_source),
    ats_token  = COALESCE(companies.ats_token,  EXCLUDED.ats_token),
    updated_at = NOW()
RETURNING *;

-- name: GetCompany :one
SELECT * FROM companies WHERE id = $1;

-- name: ListCompaniesForUser :many
SELECT c.id, c.slug, c.name, c.ats_source, c.ats_token, c.first_seen_at,
    (SELECT COUNT(*) FROM jobs j WHERE j.company_slug = c.slug) AS job_count,
    st.id AS target_id,
    COALESCE(st.enabled, FALSE) AS tracked,
    st.check_interval_minutes,
    st.last_checked_at
FROM companies c
LEFT JOIN source_targets st
    ON st.user_id = $1 AND st.source = c.ats_source AND st.value = c.ats_token
ORDER BY c.name;
