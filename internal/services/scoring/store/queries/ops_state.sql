-- name: OpsState :one
SELECT
    (SELECT count(*) FROM effect_outbox WHERE status IN ('pending', 'running')) AS outbox_pending,
    (SELECT MIN(created_at)::timestamptz FROM effect_outbox WHERE status IN ('pending', 'running')) AS oldest_pending_created_at,
    (SELECT count(*) FROM effect_outbox WHERE status = 'failed') AS outbox_failed,
    (SELECT count(*)
       FROM board_poll_state ps
       JOIN company_boards b ON b.id = ps.board_id
      WHERE b.status = 'verified' AND (ps.lease_until IS NULL OR ps.lease_until < NOW()) AND ps.next_due_at < NOW()) AS boards_overdue,
    (SELECT count(*)
       FROM board_poll_state ps
       JOIN company_boards b ON b.id = ps.board_id
      WHERE b.status = 'verified' AND ps.consecutive_failures >= 3) AS boards_failing,
    (SELECT count(*) FROM source_targets WHERE run_status = 'failed') AS source_targets_failed;

-- name: HarvestRuns :many
SELECT harvester, last_succeeded_at FROM harvest_runs;

-- name: DisabledSourceTargets :many
SELECT source, count(*) AS disabled FROM source_targets WHERE disabled_reason <> '' GROUP BY source;

-- name: UniqueRelevantJobs :many
SELECT source::text AS source, count(*) AS jobs
FROM (
    SELECT min(u.source) AS source
      FROM job_urls u
     WHERE EXISTS (SELECT 1 FROM job_scores s WHERE s.job_id = u.job_id)
     GROUP BY u.job_id
    HAVING count(DISTINCT u.source) = 1
       AND min(u.first_seen_at) > NOW() - INTERVAL '14 days'
) unique_jobs
GROUP BY source;

-- name: DiscoveryBoards :many
SELECT discovered_via::text AS via, count(*) AS boards
FROM company_boards
WHERE status = 'verified' AND discovered_via IS NOT NULL AND verified_at > NOW() - INTERVAL '14 days'
GROUP BY discovered_via;

-- name: DiscoveryRelevantJobs :many
SELECT b.discovered_via::text AS via, count(*) AS jobs
FROM jobs j
JOIN company_boards b ON b.id = j.primary_board_id
WHERE b.status = 'verified' AND b.discovered_via IS NOT NULL AND b.verified_at > NOW() - INTERVAL '14 days'
  AND EXISTS (SELECT 1 FROM job_scores s WHERE s.job_id = j.id)
GROUP BY b.discovered_via;

-- name: HarvestAdmitted :many
SELECT b.discovered_via::text AS harvester, count(DISTINCT tc.company_id) AS companies
FROM tracked_companies tc
JOIN company_boards b ON b.company_id = tc.company_id
WHERE tc.review_state IN ('new', 'kept') AND b.discovered_via IN (SELECT harvester FROM harvest_runs)
GROUP BY b.discovered_via;

-- name: EmptiedBoards :many
SELECT b.source::text AS source, count(*) AS boards
FROM board_poll_state ps
JOIN company_boards b ON b.id = ps.board_id
WHERE b.status = 'verified' AND ps.consecutive_complete_empty >= 2 AND ps.last_completed_at > NOW() - INTERVAL '7 days'
GROUP BY b.source;

-- name: UnderparsedBoards :many
SELECT b.source::text AS source, count(*) AS boards
FROM board_poll_state ps
JOIN company_boards b ON b.id = ps.board_id
WHERE b.status = 'verified' AND ps.last_reported_total > 0 AND ps.last_parsed < 0.98 * ps.last_reported_total
GROUP BY b.source;

-- name: FieldCompleteness :many
SELECT j.source::text AS source, f.field::text AS field,
       (count(*) FILTER (WHERE f.filled)::float8 / count(*))::float8 AS share
FROM jobs j
CROSS JOIN LATERAL (VALUES
    ('title', j.title <> ''),
    ('location', j.location <> ''),
    ('description', j.description <> ''),
    ('salary_raw', j.salary_raw <> ''),
    ('work_arrangement', j.work_arrangement <> '')
) AS f(field, filled)
WHERE j.closed_at IS NULL AND j.scraped_at > NOW() - INTERVAL '24 hours'
GROUP BY j.source, f.field;
