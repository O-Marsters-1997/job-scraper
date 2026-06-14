-- +goose Up
CREATE TABLE google_oauth_tokens (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id          UUID        NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    access_token_enc TEXT        NOT NULL,
    refresh_token_enc TEXT       NOT NULL,
    token_type       TEXT        NOT NULL,
    expiry           TIMESTAMPTZ,
    scope            TEXT        NOT NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
-- +goose Down
DROP TABLE google_oauth_tokens;
