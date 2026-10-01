-- +goose Up
CREATE TABLE company_profiles (
    company_id UUID        NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    source     TEXT        NOT NULL,
    data       JSONB       NOT NULL,
    fetched_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (company_id, source)
);

-- +goose Down
DROP TABLE company_profiles;
