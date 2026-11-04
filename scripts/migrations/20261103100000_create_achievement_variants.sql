-- +goose Up
CREATE TABLE achievement_variants (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    achievement_id UUID        NOT NULL REFERENCES achievements(id) ON DELETE CASCADE,
    user_id        UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    text           TEXT        NOT NULL CHECK (btrim(text) <> ''),
    label          TEXT        NOT NULL DEFAULT '',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX achievement_variants_achievement_idx ON achievement_variants (achievement_id, created_at);
CREATE INDEX achievement_variants_user_idx ON achievement_variants (user_id);

-- +goose Down
DROP TABLE achievement_variants;
