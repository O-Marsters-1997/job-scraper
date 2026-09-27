-- name: GetGoogleOAuthToken :one
SELECT access_token_enc, refresh_token_enc, token_type, expiry, scope
FROM google_oauth_tokens
WHERE user_id = $1;

-- name: UpsertGoogleOAuthToken :exec
INSERT INTO google_oauth_tokens
    (user_id, access_token_enc, refresh_token_enc, token_type, expiry, scope, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, NOW())
ON CONFLICT (user_id) DO UPDATE SET
    access_token_enc  = EXCLUDED.access_token_enc,
    refresh_token_enc = EXCLUDED.refresh_token_enc,
    token_type        = EXCLUDED.token_type,
    expiry            = EXCLUDED.expiry,
    scope             = EXCLUDED.scope,
    updated_at        = NOW();

-- name: DeleteGoogleOAuthToken :exec
DELETE FROM google_oauth_tokens WHERE user_id = $1;
