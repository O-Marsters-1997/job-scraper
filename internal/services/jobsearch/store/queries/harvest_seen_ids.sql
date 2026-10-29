-- name: ListHarvestSeenIDs :many
SELECT external_id FROM harvest_seen_ids WHERE source = $1;

-- name: RecordHarvestSeenIDs :exec
INSERT INTO harvest_seen_ids (source, external_id, outcome)
SELECT @source::text, unnest(@external_ids::text[]), @outcome::text
ON CONFLICT (source, external_id) DO UPDATE SET outcome = EXCLUDED.outcome;

-- name: MarkHarvestIDsInSitemap :exec
UPDATE harvest_seen_ids SET last_in_sitemap_at = @seen_at
WHERE source = @source AND external_id = ANY(@external_ids::text[]);

-- name: CloseJobsAbsentFromSitemap :exec
WITH closed AS (
    UPDATE jobs SET closed_at = NOW()
    WHERE jobs.source = @source AND jobs.closed_at IS NULL AND jobs.provider_posting_id IN (
        SELECT h.external_id FROM harvest_seen_ids h
        WHERE h.source = @source AND h.last_in_sitemap_at < @absent_since
    )
    RETURNING jobs.id
)
DELETE FROM option_answers WHERE job_id IN (SELECT id FROM closed);

-- name: ListHarvestCompanyCounts :many
SELECT company, live_jobs FROM harvest_company_counts WHERE source = $1;

-- name: RecordHarvestCompanyCounts :exec
INSERT INTO harvest_company_counts (source, company, live_jobs)
SELECT @source::text, unnest(@companies::text[]), unnest(@live_jobs::bigint[])
ON CONFLICT (source, company) DO UPDATE SET live_jobs = EXCLUDED.live_jobs
WHERE harvest_company_counts.live_jobs <> EXCLUDED.live_jobs;
