-- name: ListSearchConfigs :many
SELECT * FROM search_config ORDER BY user_id;

-- name: GetSearchConfig :one
SELECT * FROM search_config WHERE user_id = $1 LIMIT 1;

-- name: UpsertSearchConfig :one
INSERT INTO search_config (user_id, excluded_title_keywords, excluded_companies, excluded_seniority, excluded_locations, suitability_rubric, notify_threshold)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (user_id) DO UPDATE SET
    excluded_title_keywords = EXCLUDED.excluded_title_keywords,
    excluded_companies      = EXCLUDED.excluded_companies,
    excluded_seniority      = EXCLUDED.excluded_seniority,
    excluded_locations      = EXCLUDED.excluded_locations,
    suitability_rubric      = EXCLUDED.suitability_rubric,
    notify_threshold        = EXCLUDED.notify_threshold,
    updated_at              = NOW()
RETURNING *;
