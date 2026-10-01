-- name: GetSearchConfig :one
SELECT * FROM search_config WHERE user_id = $1 LIMIT 1;

-- name: UpsertSearchConfig :one
INSERT INTO search_config (user_id, excluded_title_keywords, excluded_companies, excluded_locations, required_locations, required_title_keywords, notify_threshold, preferences)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (user_id) DO UPDATE SET
    excluded_title_keywords = EXCLUDED.excluded_title_keywords,
    excluded_companies      = EXCLUDED.excluded_companies,
    excluded_locations      = EXCLUDED.excluded_locations,
    required_locations      = EXCLUDED.required_locations,
    required_title_keywords = EXCLUDED.required_title_keywords,
    notify_threshold        = EXCLUDED.notify_threshold,
    preferences             = EXCLUDED.preferences,
    updated_at              = NOW()
RETURNING *;
