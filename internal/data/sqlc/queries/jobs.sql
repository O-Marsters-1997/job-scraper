-- name: UpsertJob :one
INSERT INTO jobs (id, title, location, url, company_slug, source, updated_at, scraped_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
ON CONFLICT (id) DO UPDATE SET
    title        = EXCLUDED.title,
    location     = EXCLUDED.location,
    url          = EXCLUDED.url,
    updated_at   = EXCLUDED.updated_at,
    scraped_at   = NOW()
RETURNING *;

-- name: GetJobByURL :one
SELECT * FROM jobs WHERE url = $1 LIMIT 1;

-- name: ListJobs :many
SELECT * FROM jobs ORDER BY scraped_at DESC;
