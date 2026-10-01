-- name: CountLiveWTTJJobs :one
SELECT count(*) FROM wttj_jobs WHERE gone_at IS NULL;

-- name: UpsertWTTJJobs :many
INSERT INTO wttj_jobs (job_id)
SELECT DISTINCT unnest(@ids::text[])
ON CONFLICT (job_id) DO UPDATE SET last_seen_at = NOW(), gone_at = NULL
RETURNING job_id, (xmax = 0)::bool AS inserted;

-- name: InsertWTTJCompanies :many
INSERT INTO wttj_companies (url_safe_name)
SELECT DISTINCT unnest(@names::text[])
ON CONFLICT (url_safe_name) DO NOTHING
RETURNING url_safe_name;

-- name: MarkWTTJJobsGone :many
UPDATE wttj_jobs SET gone_at = NOW()
WHERE gone_at IS NULL AND NOT (job_id = ANY(@ids::text[]))
RETURNING job_id;
