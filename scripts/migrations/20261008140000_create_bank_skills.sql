-- +goose Up
CREATE TABLE bank_skills (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name       TEXT        NOT NULL,
    category   TEXT        NOT NULL DEFAULT '',
    sort_order INT         NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX bank_skills_user_name_idx ON bank_skills (user_id, lower(name));
CREATE INDEX bank_skills_user_sort_idx ON bank_skills (user_id, sort_order);

-- +goose Down
DROP TABLE bank_skills;
