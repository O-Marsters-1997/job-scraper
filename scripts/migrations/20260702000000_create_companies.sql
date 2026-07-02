-- +goose Up
-- ponytail: one row per slug. A company on two ATSes (or two companies whose
-- names slugify identically) collapse; upgrade path is UNIQUE(ats_source, ats_token)
-- plus slug aliasing if that ever bites.
CREATE TABLE companies (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    slug          TEXT        NOT NULL UNIQUE,
    name          TEXT        NOT NULL,
    ats_source    TEXT,
    ats_token     TEXT,
    first_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE companies;
