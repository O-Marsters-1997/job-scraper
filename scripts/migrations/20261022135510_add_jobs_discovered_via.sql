-- +goose Up
ALTER TABLE jobs ADD COLUMN discovered_via TEXT;

-- +goose Down
ALTER TABLE jobs DROP COLUMN discovered_via;
