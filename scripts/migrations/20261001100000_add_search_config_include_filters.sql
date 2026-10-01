-- +goose Up
ALTER TABLE search_config
    ADD COLUMN required_locations      TEXT[] NOT NULL DEFAULT '{}',
    ADD COLUMN required_title_keywords TEXT[] NOT NULL DEFAULT '{}';

-- +goose Down
ALTER TABLE search_config
    DROP COLUMN required_title_keywords,
    DROP COLUMN required_locations;
