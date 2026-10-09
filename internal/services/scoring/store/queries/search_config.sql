-- name: GetSearchConfig :one
SELECT * FROM search_config WHERE user_id = $1 LIMIT 1;

-- name: ListIncludeFilterConfigs :many
SELECT * FROM search_config
WHERE cardinality(required_locations) > 0 OR cardinality(required_title_keywords) > 0;

-- name: UpsertSearchConfig :one
INSERT INTO search_config (user_id, excluded_title_keywords, excluded_companies, excluded_locations, required_locations, required_title_keywords, notify_threshold, preferences, max_job_age_days)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
ON CONFLICT (user_id) DO UPDATE SET
    excluded_title_keywords = EXCLUDED.excluded_title_keywords,
    excluded_companies      = EXCLUDED.excluded_companies,
    excluded_locations      = EXCLUDED.excluded_locations,
    required_locations      = EXCLUDED.required_locations,
    required_title_keywords = EXCLUDED.required_title_keywords,
    notify_threshold        = EXCLUDED.notify_threshold,
    preferences             = EXCLUDED.preferences,
    max_job_age_days        = EXCLUDED.max_job_age_days,
    updated_at              = NOW()
RETURNING *;

-- name: AddExcludedCompany :one
INSERT INTO search_config (user_id, excluded_companies)
VALUES (sqlc.arg(user_id), ARRAY[sqlc.arg(name)::text])
ON CONFLICT (user_id) DO UPDATE SET
    excluded_companies = array_append(search_config.excluded_companies, sqlc.arg(name)::text),
    updated_at         = NOW()
WHERE NOT (sqlc.arg(name)::text = ANY(search_config.excluded_companies))
RETURNING user_id;

-- name: RemoveExcludedCompany :exec
UPDATE search_config
SET excluded_companies = array_remove(excluded_companies, sqlc.arg(name)::text),
    updated_at         = NOW()
WHERE user_id = sqlc.arg(user_id) AND sqlc.arg(name)::text = ANY(excluded_companies);
