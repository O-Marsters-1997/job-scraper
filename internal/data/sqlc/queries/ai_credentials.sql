-- name: UpsertUserAICredential :one
INSERT INTO user_ai_credentials (user_id, provider, api_key_enc)
VALUES ($1, $2, $3)
ON CONFLICT (user_id, provider) DO UPDATE SET
    api_key_enc = EXCLUDED.api_key_enc,
    updated_at  = now()
RETURNING *;

-- name: GetUserAICredential :one
SELECT * FROM user_ai_credentials WHERE user_id = $1 AND provider = $2 LIMIT 1;

-- name: DeleteUserAICredential :exec
DELETE FROM user_ai_credentials WHERE user_id = $1 AND provider = $2;

-- name: ListUserAICredentialProviders :many
SELECT provider FROM user_ai_credentials WHERE user_id = $1 ORDER BY provider;

-- name: ListUsersWithProvider :many
SELECT user_id FROM user_ai_credentials WHERE provider = $1 ORDER BY user_id;
