-- name: InsertScrapeRun :exec
INSERT INTO scrape_runs (id, kind, source, source_target_id, trigger)
VALUES (sqlc.arg('id'), 'search', sqlc.arg('source'), sqlc.arg('source_target_id'), sqlc.arg('trigger'));

-- name: ClassifyScrapeCard :one
SELECT CASE
    WHEN EXISTS (
        SELECT 1 FROM candidate_discoveries d
        JOIN job_candidates c ON c.id = d.candidate_id
        WHERE c.normalized_url = sqlc.arg('normalized_url') AND d.source_target_id = sqlc.arg('source_target_id')
    ) THEN 'repeat'
    WHEN EXISTS (SELECT 1 FROM job_urls WHERE normalized_url = sqlc.arg('normalized_url'))
      OR EXISTS (
        SELECT 1 FROM candidate_discoveries d
        JOIN job_candidates c ON c.id = d.candidate_id
        WHERE c.normalized_url = sqlc.arg('normalized_url') AND d.source_target_id <> sqlc.arg('source_target_id')
    ) THEN 'elsewhere'
    ELSE 'new'
END::text;

-- name: AddScrapeRunCards :exec
UPDATE scrape_runs SET
    listings        = listings + sqlc.arg('listings'),
    new_listings    = new_listings + sqlc.arg('new_listings'),
    elsewhere       = elsewhere + sqlc.arg('elsewhere'),
    repeat_listings = repeat_listings + sqlc.arg('repeat_listings')
WHERE id = sqlc.arg('id');

-- name: ScrapeRunPageDone :exec
UPDATE scrape_runs SET pages = pages + 1 WHERE id = $1;

-- name: ScrapeRunDeferred :exec
UPDATE scrape_runs SET deferred = deferred + 1 WHERE id = $1;

-- name: ScrapeRunAttemptFailed :exec
WITH bumped AS (
    UPDATE scrape_runs SET retries = retries + 1 WHERE scrape_runs.id = sqlc.arg('run_id') RETURNING scrape_runs.id
)
INSERT INTO scrape_run_errors (run_id, page, attempt, message)
SELECT bumped.id, sqlc.arg('page'), sqlc.arg('attempt'), sqlc.arg('message') FROM bumped;

-- name: FinishScrapeRun :exec
UPDATE scrape_runs SET finished_at = NOW(), stop_reason = $2
WHERE id = $1 AND finished_at IS NULL;

-- name: DeleteExpiredScrapeRuns :exec
DELETE FROM scrape_runs WHERE started_at < NOW() - INTERVAL '30 days';

-- name: ListScrapeRuns :many
SELECT r.id, r.kind, r.source, r.source_target_id, t.value AS target_value, t.filters AS target_filters,
       r.trigger, r.started_at, r.finished_at, r.stop_reason,
       r.pages, r.listings, r.new_listings, r.elsewhere, r.repeat_listings, r.deferred, r.retries
FROM scrape_runs r
JOIN source_targets t ON t.id = r.source_target_id
WHERE t.user_id = sqlc.arg('user_id')
  AND (sqlc.narg('target_id')::uuid IS NULL OR r.source_target_id = sqlc.narg('target_id'))
  AND (sqlc.narg('from_at')::timestamptz IS NULL OR r.started_at >= sqlc.narg('from_at'))
  AND (sqlc.narg('to_at')::timestamptz IS NULL OR r.started_at <= sqlc.narg('to_at'))
  AND (sqlc.narg('q')::text IS NULL
       OR t.value ILIKE '%' || sqlc.narg('q') || '%'
       OR EXISTS (SELECT 1 FROM scrape_run_errors e WHERE e.run_id = r.id AND e.message ILIKE '%' || sqlc.narg('q') || '%'))
  AND (COALESCE(cardinality(sqlc.arg('outcomes')::text[]), 0) = 0 OR (CASE WHEN r.finished_at IS NULL THEN 'running' WHEN r.stop_reason = 'failed' THEN 'failed' WHEN r.retries > 0 THEN 'retried' ELSE 'succeeded' END) = ANY(sqlc.arg('outcomes')::text[]))
  AND (COALESCE(cardinality(sqlc.arg('kinds')::text[]), 0) = 0 OR r.kind = ANY(sqlc.arg('kinds')::text[]))
  AND (COALESCE(cardinality(sqlc.arg('sources')::text[]), 0) = 0 OR r.source = ANY(sqlc.arg('sources')::text[]))
  AND (COALESCE(cardinality(sqlc.arg('triggers')::text[]), 0) = 0 OR r.trigger = ANY(sqlc.arg('triggers')::text[]))
  AND (sqlc.narg('before_started_at')::timestamptz IS NULL
       OR (r.started_at, r.id) < (sqlc.narg('before_started_at')::timestamptz, sqlc.narg('before_id')::uuid))
ORDER BY r.started_at DESC, r.id DESC
LIMIT sqlc.arg('row_limit');

-- name: ListScrapeRunErrors :many
SELECT run_id, page, attempt, message, at
FROM scrape_run_errors
WHERE run_id = ANY(sqlc.arg('run_ids')::uuid[])
ORDER BY run_id, id;

-- name: SummariseScrapeRuns :many
SELECT (CASE WHEN r.finished_at IS NULL THEN 'running' WHEN r.stop_reason = 'failed' THEN 'failed' WHEN r.retries > 0 THEN 'retried' ELSE 'succeeded' END)::text AS outcome, r.kind, r.source, r.trigger,
       date_bin(make_interval(secs => sqlc.arg('bucket_secs')::float8), r.started_at, TIMESTAMPTZ '1970-01-01')::timestamptz AS bucket,
       COUNT(*)::int AS runs
FROM scrape_runs r
JOIN source_targets t ON t.id = r.source_target_id
WHERE t.user_id = sqlc.arg('user_id')
  AND (sqlc.narg('target_id')::uuid IS NULL OR r.source_target_id = sqlc.narg('target_id'))
  AND (sqlc.narg('from_at')::timestamptz IS NULL OR r.started_at >= sqlc.narg('from_at'))
  AND (sqlc.narg('to_at')::timestamptz IS NULL OR r.started_at <= sqlc.narg('to_at'))
  AND (sqlc.narg('q')::text IS NULL
       OR t.value ILIKE '%' || sqlc.narg('q') || '%'
       OR EXISTS (SELECT 1 FROM scrape_run_errors e WHERE e.run_id = r.id AND e.message ILIKE '%' || sqlc.narg('q') || '%'))
GROUP BY 1, 2, 3, 4, 5;
