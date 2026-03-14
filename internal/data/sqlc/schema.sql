CREATE TABLE IF NOT EXISTS jobs (
    id           UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    title        TEXT         NOT NULL,
    location     TEXT         NOT NULL DEFAULT '',
    url          TEXT         NOT NULL UNIQUE,
    company_slug TEXT         NOT NULL,
    source       TEXT         NOT NULL,
    updated_at   TIMESTAMPTZ  NOT NULL,
    scraped_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
