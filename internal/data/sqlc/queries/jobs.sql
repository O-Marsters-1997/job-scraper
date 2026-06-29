-- name: UpsertJob :one
INSERT INTO jobs (title, location, url, company_slug, source, updated_at, scraped_at, description, salary_raw, work_arrangement)
VALUES ($1, $2, $3, $4, $5, $6, NOW(), $7, $8, $9)
ON CONFLICT (url) DO UPDATE SET
    title            = EXCLUDED.title,
    location         = EXCLUDED.location,
    updated_at       = EXCLUDED.updated_at,
    scraped_at       = NOW(),
    description      = EXCLUDED.description,
    salary_raw       = EXCLUDED.salary_raw,
    work_arrangement = EXCLUDED.work_arrangement
RETURNING *;

-- name: UpsertJobs :batchexec
INSERT INTO jobs (title, location, url, company_slug, source, updated_at, scraped_at, description, salary_raw, work_arrangement)
VALUES ($1, $2, $3, $4, $5, $6, NOW(), $7, $8, $9)
ON CONFLICT (url) DO UPDATE SET
    title            = EXCLUDED.title,
    location         = EXCLUDED.location,
    updated_at       = EXCLUDED.updated_at,
    scraped_at       = NOW(),
    description      = EXCLUDED.description,
    salary_raw       = EXCLUDED.salary_raw,
    work_arrangement = EXCLUDED.work_arrangement;

-- name: GetJobByURL :one
SELECT * FROM jobs WHERE url = $1 LIMIT 1;

-- name: ListJobs :many
SELECT j.id, j.title, j.location, j.url, j.company_slug, j.source, j.updated_at, j.scraped_at, j.description, j.salary_raw, j.work_arrangement, js.relevance_score, js.suitability_score, js.reasoning, js.matched, js.missing, COALESCE(js.suitability_skipped, false) AS suitability_skipped
FROM jobs j
LEFT JOIN job_scores js ON js.job_id = j.id AND js.user_id = $1
ORDER BY COALESCE(js.suitability_score, -1) DESC, j.scraped_at DESC;

-- name: GetJob :one
SELECT j.id, j.title, j.location, j.url, j.company_slug, j.source, j.updated_at, j.scraped_at, j.description, j.salary_raw, j.work_arrangement, js.relevance_score, js.suitability_score, js.reasoning, js.matched, js.missing, COALESCE(js.suitability_skipped, false) AS suitability_skipped
FROM jobs j
LEFT JOIN job_scores js ON js.job_id = j.id AND js.user_id = $2
WHERE j.id = $1
LIMIT 1;

-- name: ExistingURLs :many
SELECT url FROM jobs WHERE url = ANY($1::text[]);

-- name: ListJobsSince :many
SELECT * FROM jobs WHERE scraped_at > $1 ORDER BY scraped_at DESC;
