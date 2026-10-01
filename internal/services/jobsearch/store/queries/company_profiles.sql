-- name: UpsertCompanyProfile :exec
INSERT INTO company_profiles (company_id, source, data) VALUES ($1, $2, $3)
ON CONFLICT (company_id, source) DO UPDATE SET data = EXCLUDED.data, fetched_at = NOW();

-- name: ListNewCompanyProfiles :many
SELECT DISTINCT ON (cp.company_id) cp.company_id, cp.data
FROM tracked_companies tc
JOIN company_profiles cp ON cp.company_id = tc.company_id
WHERE tc.user_id = $1 AND tc.review_state = 'new'
ORDER BY cp.company_id, cp.fetched_at DESC;
