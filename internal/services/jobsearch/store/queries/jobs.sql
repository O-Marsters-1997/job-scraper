-- name: ListJobs :many
SELECT j.id, j.title, j.location, j.url, j.company_slug, j.source, j.updated_at, j.scraped_at, j.salary_raw, j.work_arrangement, j.company_id, j.primary_board_id, j.provider_posting_id, j.content_fingerprint, js.suitability_score, js.band, js.breakdown, COALESCE(jg.grade, '')::text AS grade, (jv.job_id IS NOT NULL)::bool AS seen
FROM jobs j
JOIN job_scores js ON js.job_id = j.id AND js.user_id = $1
LEFT JOIN job_grades jg ON jg.job_id = j.id AND jg.user_id = $1
LEFT JOIN job_views jv ON jv.job_id = j.id AND jv.user_id = $1
WHERE j.closed_at IS NULL
  AND j.updated_at > now() - interval '90 days'
  AND NOT js.breakdown @> '[{"effect":"blocked"}]'::jsonb
  AND NOT EXISTS (SELECT 1 FROM job_grades g WHERE g.job_id = j.id AND g.user_id = $1 AND g.grade = 'no')
ORDER BY js.suitability_score DESC, j.scraped_at DESC;

-- name: PageJobs :many
SELECT j.id, j.title, j.location, j.url, j.company_slug, j.source, j.updated_at, j.scraped_at, j.salary_raw, j.work_arrangement, j.company_id, j.primary_board_id, j.provider_posting_id, j.content_fingerprint, js.suitability_score, js.band, js.breakdown, COALESCE(jg.grade, '')::text AS grade, (jv.job_id IS NOT NULL)::bool AS seen
FROM jobs j
LEFT JOIN job_scores js ON js.job_id = j.id AND js.user_id = sqlc.arg(user_id)::uuid
LEFT JOIN job_grades jg ON jg.job_id = j.id AND jg.user_id = sqlc.arg(user_id)::uuid
LEFT JOIN job_views jv ON jv.job_id = j.id AND jv.user_id = sqlc.arg(user_id)::uuid
WHERE (sqlc.narg(cursor_time)::timestamptz IS NULL OR (j.scraped_at, j.id) < (sqlc.narg(cursor_time)::timestamptz, sqlc.narg(cursor_id)::uuid))
  AND (sqlc.narg(company_id)::uuid IS NULL OR j.company_id = sqlc.narg(company_id)::uuid OR (j.company_id IS NULL AND j.company_slug = (SELECT slug FROM companies WHERE id = sqlc.narg(company_id)::uuid)))
  AND (sqlc.arg(availability)::text = 'all' OR (sqlc.arg(availability)::text = 'open' AND j.closed_at IS NULL) OR (sqlc.arg(availability)::text = 'closed' AND j.closed_at IS NOT NULL))
  AND (NOT sqlc.arg(scored_only)::bool OR EXISTS (SELECT 1 FROM job_scores s WHERE s.job_id = j.id AND s.user_id = sqlc.arg(user_id)::uuid))
  AND (sqlc.arg(since_days)::int = 0 OR j.updated_at >= now() - make_interval(days => sqlc.arg(since_days)::int))
  AND NOT COALESCE(js.breakdown @> '[{"effect":"blocked"}]'::jsonb, false)
  AND NOT EXISTS (SELECT 1 FROM job_grades g WHERE g.job_id = j.id AND g.user_id = sqlc.arg(user_id)::uuid AND g.grade = 'no')
ORDER BY j.scraped_at DESC, j.id DESC
LIMIT sqlc.arg(page_limit)::int;

-- name: GetJob :one
SELECT j.id, j.title, j.location, j.url, j.company_slug, j.source, j.updated_at, j.scraped_at, j.description, j.salary_raw, j.work_arrangement, j.company_id, j.primary_board_id, j.provider_posting_id, j.content_fingerprint, js.suitability_score, js.band, js.breakdown, (jv.job_id IS NOT NULL)::bool AS seen
FROM jobs j
LEFT JOIN job_scores js ON js.job_id = j.id AND js.user_id = $2
LEFT JOIN job_views jv ON jv.job_id = j.id AND jv.user_id = $2
WHERE j.id = $1
LIMIT 1;

-- name: ExistingURLs :many
SELECT url FROM jobs WHERE url = ANY($1::text[])
UNION
SELECT normalized_url FROM job_candidates WHERE normalized_url = ANY($1::text[]);

-- name: MarkJobsSeen :exec
INSERT INTO job_views (user_id, job_id)
SELECT sqlc.arg(user_id)::uuid, j.id
FROM jobs j
WHERE j.id = ANY(sqlc.arg(job_ids)::uuid[])
ON CONFLICT DO NOTHING;

-- name: MarkJobsUnseen :exec
DELETE FROM job_views
WHERE user_id = sqlc.arg(user_id)::uuid AND job_id = ANY(sqlc.arg(job_ids)::uuid[]);
