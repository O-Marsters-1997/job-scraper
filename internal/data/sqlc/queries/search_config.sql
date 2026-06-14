-- name: GetSearchConfig :one
SELECT * FROM search_config WHERE user_id = $1 LIMIT 1;

-- name: UpsertSearchConfig :one
INSERT INTO search_config (user_id, role, location, keywords, suitability_rubric, relevance_cutoff, notify_threshold)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (user_id) DO UPDATE SET
    role               = EXCLUDED.role,
    location           = EXCLUDED.location,
    keywords           = EXCLUDED.keywords,
    suitability_rubric = EXCLUDED.suitability_rubric,
    relevance_cutoff   = EXCLUDED.relevance_cutoff,
    notify_threshold   = EXCLUDED.notify_threshold,
    updated_at         = NOW()
RETURNING *;
