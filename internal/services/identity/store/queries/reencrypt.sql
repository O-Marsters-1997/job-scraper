-- name: ListLegacyAICredentialIDs :many
SELECT id FROM user_ai_credentials WHERE api_key_enc NOT LIKE 'v1:%' ORDER BY id LIMIT 500;

-- name: LockAICredential :one
SELECT user_id, provider, api_key_enc FROM user_ai_credentials WHERE id = $1 FOR UPDATE;

-- name: SetAICredentialEnc :exec
UPDATE user_ai_credentials SET api_key_enc = $2 WHERE id = $1;

-- name: ListLegacyGoogleTokenIDs :many
SELECT id FROM google_oauth_tokens
WHERE access_token_enc NOT LIKE 'v1:%' OR refresh_token_enc NOT LIKE 'v1:%'
ORDER BY id LIMIT 500;

-- name: LockGoogleToken :one
SELECT user_id, access_token_enc, refresh_token_enc FROM google_oauth_tokens WHERE id = $1 FOR UPDATE;

-- name: SetGoogleTokenEnc :exec
UPDATE google_oauth_tokens SET access_token_enc = $2, refresh_token_enc = $3 WHERE id = $1;
