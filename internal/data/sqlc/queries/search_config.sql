-- name: ListSearchConfigs :many
SELECT * FROM search_config ORDER BY user_id;

-- name: GetSearchConfig :one
SELECT * FROM search_config WHERE user_id = $1 LIMIT 1;

-- name: UpsertSearchConfig :one
INSERT INTO search_config (user_id, excluded_title_keywords, excluded_companies, excluded_seniority, excluded_locations, notify_threshold, scoring_questions)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (user_id) DO UPDATE SET
    excluded_title_keywords = EXCLUDED.excluded_title_keywords,
    excluded_companies      = EXCLUDED.excluded_companies,
    excluded_seniority      = EXCLUDED.excluded_seniority,
    excluded_locations      = EXCLUDED.excluded_locations,
    notify_threshold        = EXCLUDED.notify_threshold,
    scoring_questions       = EXCLUDED.scoring_questions,
    updated_at              = NOW()
RETURNING *;
