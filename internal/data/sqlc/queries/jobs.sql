-- name: UpsertJob :one
INSERT INTO jobs (title, location, url, company_slug, source, updated_at, scraped_at)
VALUES ($1, $2, $3, $4, $5, $6, NOW())
ON CONFLICT (url) DO UPDATE SET
    title      = EXCLUDED.title,
    location   = EXCLUDED.location,
    updated_at = EXCLUDED.updated_at,
    scraped_at = NOW()
RETURNING *;

-- name: UpsertJobs :batchexec
INSERT INTO jobs (title, location, url, company_slug, source, updated_at, scraped_at)
VALUES ($1, $2, $3, $4, $5, $6, NOW())
ON CONFLICT (url) DO UPDATE SET
    title      = EXCLUDED.title,
    location   = EXCLUDED.location,
    updated_at = EXCLUDED.updated_at,
    scraped_at = NOW();

-- name: GetJobByURL :one
SELECT * FROM jobs WHERE url = $1 LIMIT 1;

-- name: ListJobs :many
SELECT * FROM jobs ORDER BY scraped_at DESC;

-- name: ExistingURLs :many
SELECT url FROM jobs WHERE url = ANY($1::text[]);

-- name: ListJobsSince :many
SELECT * FROM jobs WHERE scraped_at > $1 ORDER BY scraped_at DESC;
