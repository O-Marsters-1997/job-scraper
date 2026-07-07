-- +goose Up
ALTER TABLE search_config
    ADD COLUMN excluded_title_keywords TEXT[] NOT NULL DEFAULT '{}',
    ADD COLUMN excluded_companies      TEXT[] NOT NULL DEFAULT '{}',
    ADD COLUMN excluded_seniority      TEXT[] NOT NULL DEFAULT '{}',
    ADD COLUMN excluded_locations      TEXT[] NOT NULL DEFAULT '{}',
    DROP COLUMN role,
    DROP COLUMN location,
    DROP COLUMN keywords,
    DROP COLUMN relevance_cutoff;

-- +goose Down
ALTER TABLE search_config
    DROP COLUMN excluded_title_keywords,
    DROP COLUMN excluded_companies,
    DROP COLUMN excluded_seniority,
    DROP COLUMN excluded_locations,
    ADD COLUMN role             TEXT   NOT NULL DEFAULT '',
    ADD COLUMN location         TEXT   NOT NULL DEFAULT '',
    ADD COLUMN keywords         TEXT[] NOT NULL DEFAULT '{}',
    ADD COLUMN relevance_cutoff INT    NOT NULL DEFAULT 0;
