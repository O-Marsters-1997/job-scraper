-- +goose Up
UPDATE jobs j SET company_id = c.id
FROM companies c
WHERE j.company_id IS NULL AND j.company_slug = c.slug;

-- +goose Down
SELECT 1;
