-- +goose Up
WITH candidates AS (
    SELECT id, slug AS old_slug, btrim(regexp_replace(slug, '-{2,}', '-', 'g'), '-') AS new_slug
    FROM companies
    WHERE slug ~ '--' OR slug ~ '^-|-$'
),
winners AS (
    SELECT DISTINCT ON (new_slug) id, old_slug, new_slug
    FROM candidates c
    WHERE new_slug <> ''
      AND NOT EXISTS (SELECT 1 FROM companies o WHERE o.slug = c.new_slug)
    ORDER BY new_slug, id
),
renamed AS (
    UPDATE companies c SET slug = w.new_slug, updated_at = NOW()
    FROM winners w WHERE c.id = w.id
    RETURNING c.id, w.old_slug, w.new_slug
)
UPDATE jobs j SET company_slug = r.new_slug
FROM renamed r
WHERE j.company_id = r.id AND j.company_slug = r.old_slug;

-- +goose Down
SELECT 1;
