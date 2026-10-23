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
