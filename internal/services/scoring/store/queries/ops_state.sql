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
