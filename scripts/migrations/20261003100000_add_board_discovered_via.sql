-- +goose Up
ALTER TABLE company_boards ADD COLUMN discovered_via TEXT;

-- +goose Down
ALTER TABLE company_boards DROP COLUMN discovered_via;
