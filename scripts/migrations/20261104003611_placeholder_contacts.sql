-- +goose Up
ALTER TABLE contacts ALTER COLUMN name DROP NOT NULL;

-- +goose Down
UPDATE contacts SET name = 'Unknown' WHERE name IS NULL;
ALTER TABLE contacts ALTER COLUMN name SET NOT NULL;
