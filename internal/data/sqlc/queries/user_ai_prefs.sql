-- name: GetUserAIPrefs :one
SELECT * FROM user_ai_prefs WHERE user_id = $1 LIMIT 1;

-- name: UpsertUserAIPrefs :one
INSERT INTO user_ai_prefs (user_id, suitability_model)
VALUES ($1, $2)
ON CONFLICT (user_id) DO UPDATE SET
    suitability_model = EXCLUDED.suitability_model,
    updated_at        = now()
RETURNING *;
