-- name: GetCompanyProfile :one
SELECT data FROM company_profiles WHERE company_id = $1 ORDER BY fetched_at DESC LIMIT 1;
