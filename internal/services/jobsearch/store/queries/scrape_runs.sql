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
SELECT r.id, r.kind, r.source, r.source_target_id, COALESCE(t.value, '')::text AS target_value, t.filters AS target_filters,
       r.board_id, b.company_id, COALESCE(c.name, '')::text AS company_name,
       r.trigger, r.started_at, r.finished_at, r.stop_reason,
       r.pages, r.listings, r.new_listings, r.elsewhere, r.repeat_listings, r.removed, r.reopened, r.deferred, r.retries
FROM scrape_runs r
LEFT JOIN source_targets t ON t.id = r.source_target_id
LEFT JOIN company_boards b ON b.id = r.board_id
LEFT JOIN companies c ON c.id = b.company_id
WHERE (t.user_id = sqlc.arg('user_id')
       OR EXISTS (SELECT 1 FROM tracked_companies tc WHERE tc.user_id = sqlc.arg('user_id') AND tc.company_id = b.company_id))
  AND (sqlc.narg('target_id')::uuid IS NULL OR r.source_target_id = sqlc.narg('target_id'))
  AND (sqlc.narg('company_id')::uuid IS NULL OR b.company_id = sqlc.narg('company_id'))
  AND (sqlc.narg('from_at')::timestamptz IS NULL OR r.started_at >= sqlc.narg('from_at'))
  AND (sqlc.narg('to_at')::timestamptz IS NULL OR r.started_at <= sqlc.narg('to_at'))
  AND (sqlc.narg('q')::text IS NULL
       OR t.value ILIKE '%' || sqlc.narg('q') || '%'
       OR c.name ILIKE '%' || sqlc.narg('q') || '%'
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

-- name: ListSourceHealthRuns :many
SELECT t.id AS source_target_id, t.source, t.value AS target_value, t.filters AS target_filters,
       r.started_at, r.stop_reason,
       COALESCE(r.new_listings, 0)::int AS new_listings, COALESCE(r.elsewhere, 0)::int AS elsewhere,
       COALESCE(r.retries, 0)::int AS retries
FROM source_targets t
LEFT JOIN LATERAL (
    SELECT id, started_at, stop_reason, new_listings, elsewhere, retries
    FROM scrape_runs
    WHERE source_target_id = t.id AND finished_at IS NOT NULL
    ORDER BY started_at DESC, id DESC
    LIMIT 5
) r ON TRUE
WHERE t.user_id = sqlc.arg('user_id')
ORDER BY t.id, r.started_at, r.id;

-- name: SummariseScrapeRuns :many
SELECT (CASE WHEN r.finished_at IS NULL THEN 'running' WHEN r.stop_reason = 'failed' THEN 'failed' WHEN r.retries > 0 THEN 'retried' ELSE 'succeeded' END)::text AS outcome, r.kind, r.source, r.trigger,
       date_bin(make_interval(secs => sqlc.arg('bucket_secs')::float8), r.started_at, TIMESTAMPTZ '1970-01-01')::timestamptz AS bucket,
       COUNT(*)::int AS runs,
       COALESCE(SUM(r.new_listings), 0)::int AS new_listings, COALESCE(SUM(r.removed), 0)::int AS removed,
       MAX(r.started_at)::timestamptz AS last_started_at
FROM scrape_runs r
LEFT JOIN source_targets t ON t.id = r.source_target_id
LEFT JOIN company_boards b ON b.id = r.board_id
LEFT JOIN companies c ON c.id = b.company_id
WHERE (t.user_id = sqlc.arg('user_id')
       OR EXISTS (SELECT 1 FROM tracked_companies tc WHERE tc.user_id = sqlc.arg('user_id') AND tc.company_id = b.company_id))
  AND (sqlc.narg('target_id')::uuid IS NULL OR r.source_target_id = sqlc.narg('target_id'))
  AND (sqlc.narg('company_id')::uuid IS NULL OR b.company_id = sqlc.narg('company_id'))
  AND (sqlc.narg('from_at')::timestamptz IS NULL OR r.started_at >= sqlc.narg('from_at'))
  AND (sqlc.narg('to_at')::timestamptz IS NULL OR r.started_at <= sqlc.narg('to_at'))
  AND (sqlc.narg('q')::text IS NULL
       OR t.value ILIKE '%' || sqlc.narg('q') || '%'
       OR c.name ILIKE '%' || sqlc.narg('q') || '%'
       OR EXISTS (SELECT 1 FROM scrape_run_errors e WHERE e.run_id = r.id AND e.message ILIKE '%' || sqlc.narg('q') || '%'))
GROUP BY 1, 2, 3, 4, 5;

-- name: BoardRunAttemptFailed :exec
WITH run AS (
    INSERT INTO scrape_runs (id, kind, source, board_id, trigger, started_at, retries)
    VALUES (sqlc.arg('id'), 'board', sqlc.arg('source'), sqlc.arg('board_id'), sqlc.arg('trigger'), sqlc.arg('started_at'), 1)
    ON CONFLICT (id) DO UPDATE SET retries = scrape_runs.retries + 1
    RETURNING id
)
INSERT INTO scrape_run_errors (run_id, page, attempt, message)
SELECT run.id, 0, sqlc.arg('attempt'), sqlc.arg('message') FROM run;

-- name: FinishBoardRun :execrows
INSERT INTO scrape_runs (id, kind, source, board_id, trigger, started_at, finished_at, stop_reason, listings, new_listings, elsewhere, removed, reopened)
VALUES (sqlc.arg('id'), 'board', sqlc.arg('source'), sqlc.arg('board_id'), sqlc.arg('trigger'), sqlc.arg('started_at'), NOW(), sqlc.arg('stop_reason'),
        sqlc.arg('listings'), sqlc.arg('new_listings'), sqlc.arg('elsewhere'), sqlc.arg('removed'), sqlc.arg('reopened'))
ON CONFLICT (id) DO UPDATE SET
    finished_at  = NOW(),
    stop_reason  = EXCLUDED.stop_reason,
    listings     = EXCLUDED.listings,
    new_listings = EXCLUDED.new_listings,
    elsewhere    = EXCLUDED.elsewhere,
    removed      = EXCLUDED.removed,
    reopened     = EXCLUDED.reopened
WHERE scrape_runs.finished_at IS NULL;

-- name: InsertScrapeRunJobChanges :exec
INSERT INTO scrape_run_job_changes (run_id, job_id, change, snapshot)
SELECT sqlc.arg('run_id')::uuid, t.job_id, t.change, t.snapshot
FROM jsonb_to_recordset(sqlc.arg('changes')::jsonb) AS t(job_id uuid, change text, snapshot jsonb);

-- name: ListScrapeRunJobChanges :many
SELECT run_id, job_id, change, (snapshot->>'Title')::text AS title
FROM scrape_run_job_changes
WHERE run_id = ANY(sqlc.arg('run_ids')::uuid[])
ORDER BY run_id, change, title, job_id;

-- name: ListScrapeRunJobSnapshots :many
SELECT job_id, change, snapshot
FROM scrape_run_job_changes
WHERE run_id = sqlc.arg('run_id')
ORDER BY change, snapshot->>'Title', job_id
LIMIT 1000;

-- name: ScrapeRunTracked :one
SELECT EXISTS (
  SELECT 1 FROM scrape_runs r
  JOIN company_boards b ON b.id = r.board_id
  JOIN tracked_companies tc ON tc.company_id = b.company_id AND tc.user_id = sqlc.arg('user_id')
  WHERE r.id = sqlc.arg('run_id')
)::bool;

-- name: ListBoardHealthRuns :many
SELECT b.id AS board_id, c.name AS company_name, b.source,
       r.started_at, r.stop_reason,
       COALESCE(r.new_listings, 0)::int AS new_listings, COALESCE(r.elsewhere, 0)::int AS elsewhere,
       COALESCE(r.removed, 0)::int AS removed, COALESCE(r.retries, 0)::int AS retries
FROM company_boards b
JOIN companies c ON c.id = b.company_id
LEFT JOIN LATERAL (
    SELECT id, started_at, stop_reason, new_listings, elsewhere, removed, retries
    FROM scrape_runs
    WHERE board_id = b.id AND finished_at IS NOT NULL
    ORDER BY started_at DESC, id DESC
    LIMIT 5
) r ON TRUE
WHERE b.status = 'verified'
  AND EXISTS (SELECT 1 FROM tracked_companies tc WHERE tc.user_id = sqlc.arg('user_id') AND tc.company_id = b.company_id)
ORDER BY b.id, r.started_at, r.id;

-- name: ListRecentNewJobSnapshots :many
SELECT r.source_target_id, r.board_id, (ch.snapshot->>'Title')::text AS title,
       (ch.snapshot->>'Location')::text AS location, (ch.snapshot->>'CompanySlug')::text AS company_slug
FROM scrape_run_job_changes ch
JOIN scrape_runs r ON r.id = ch.run_id
LEFT JOIN source_targets t ON t.id = r.source_target_id
LEFT JOIN company_boards b ON b.id = r.board_id
WHERE ch.change = 'new'
  AND r.started_at >= sqlc.arg('since')::timestamptz
  AND (t.user_id = sqlc.arg('user_id')
       OR EXISTS (SELECT 1 FROM tracked_companies tc WHERE tc.user_id = sqlc.arg('user_id') AND tc.company_id = b.company_id))
UNION ALL
SELECT t.id, NULL::uuid, c.card_title, c.card_location, c.card_company
FROM source_targets t
JOIN candidate_discoveries d ON d.source_target_id = t.id
JOIN job_candidates c ON c.id = d.candidate_id
WHERE t.user_id = sqlc.arg('user_id')
  AND c.first_seen_at >= sqlc.arg('since')::timestamptz
  AND EXISTS (SELECT 1 FROM scrape_runs r
              WHERE r.source_target_id = t.id AND c.first_seen_at BETWEEN r.started_at AND COALESCE(r.finished_at, NOW()))
LIMIT 50000;
