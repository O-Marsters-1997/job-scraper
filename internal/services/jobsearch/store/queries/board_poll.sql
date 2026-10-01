-- name: ListDueBoards :many
SELECT b.id, b.company_id, c.slug AS company_slug, b.source, b.board_token,
       MIN(tc.check_interval_minutes)::int AS interval_minutes
FROM company_boards b
JOIN companies c ON c.id = b.company_id
JOIN tracked_companies tc ON tc.company_id = b.company_id AND tc.enabled
LEFT JOIN board_poll_state s ON s.board_id = b.id
WHERE b.status = 'verified'
GROUP BY b.id, c.slug, s.last_scheduled_at, s.lease_until, s.next_due_at
HAVING (s.last_scheduled_at IS NULL OR s.last_scheduled_at + MIN(tc.check_interval_minutes) * INTERVAL '1 minute' <= NOW())
   AND (s.lease_until IS NULL OR s.lease_until < NOW())
   AND (s.next_due_at IS NULL OR s.next_due_at <= NOW())
ORDER BY b.id;

-- name: ListActiveBoards :many
SELECT b.id, b.company_id, c.slug AS company_slug, b.source, b.board_token,
       MIN(tc.check_interval_minutes)::int AS interval_minutes
FROM company_boards b
JOIN companies c ON c.id = b.company_id
JOIN tracked_companies tc ON tc.company_id = b.company_id AND tc.enabled
WHERE b.status = 'verified'
GROUP BY b.id, c.slug
ORDER BY b.id;

-- name: GetPollBoard :one
SELECT b.id, b.company_id, c.slug AS company_slug, b.source, b.board_token,
       MIN(tc.check_interval_minutes)::int AS interval_minutes, s.last_scheduled_at
FROM company_boards b
JOIN companies c ON c.id = b.company_id
JOIN tracked_companies tc ON tc.company_id = b.company_id AND tc.enabled
LEFT JOIN board_poll_state s ON s.board_id = b.id
WHERE b.id = $1 AND b.status = 'verified'
GROUP BY b.id, c.slug, s.last_scheduled_at;

-- name: EnsureBoardPollState :exec
INSERT INTO board_poll_state (board_id)
SELECT id FROM company_boards WHERE id = $1 AND status = 'verified'
ON CONFLICT (board_id) DO NOTHING;

-- name: ClaimPollState :one
UPDATE board_poll_state
SET lease_owner = $2, lease_until = NOW() + INTERVAL '30 minutes',
    last_started_at = NOW(), last_snapshot_version = last_snapshot_version + 1
WHERE board_id = $1 AND (lease_until IS NULL OR lease_until < NOW())
  AND (sqlc.arg(manual)::boolean OR next_due_at <= NOW() AND (last_scheduled_at IS NULL OR last_scheduled_at + (
      SELECT MIN(tc.check_interval_minutes) * INTERVAL '1 minute'
      FROM company_boards b JOIN tracked_companies tc ON tc.company_id = b.company_id AND tc.enabled
      WHERE b.id = board_poll_state.board_id
  ) <= NOW()))
  AND EXISTS (
      SELECT 1 FROM company_boards b
      JOIN tracked_companies tc ON tc.company_id = b.company_id AND tc.enabled
      WHERE b.id = board_poll_state.board_id AND b.status = 'verified'
  )
RETURNING last_snapshot_version, last_started_at;

-- name: LockPollState :one
SELECT * FROM board_poll_state WHERE board_id = $1 FOR UPDATE;

-- name: FailPollState :execrows
UPDATE board_poll_state
SET lease_owner = NULL, lease_until = NULL, consecutive_failures = consecutive_failures + 1
WHERE board_id = $1 AND lease_owner = $2 AND last_snapshot_version = $3;

-- name: FindBoardJobAliases :many
SELECT normalized_url, job_id FROM job_urls WHERE normalized_url = ANY($1::text[]);

-- name: ObserveBoardJob :exec
INSERT INTO board_job_observations (board_id, job_id, last_seen_at, last_snapshot_version)
VALUES ($1, $2, NOW(), $3)
ON CONFLICT (board_id, job_id) DO UPDATE
SET last_seen_at = EXCLUDED.last_seen_at, last_snapshot_version = EXCLUDED.last_snapshot_version;

-- name: ReopenObservedBoardJobs :exec
UPDATE jobs SET closed_at = NULL
WHERE id IN (SELECT job_id FROM board_job_observations WHERE board_id = $1 AND last_snapshot_version = $2)
  AND closed_at IS NOT NULL;

-- name: CloseMissingBoardJobs :exec
-- Writes scoring's option_answers table directly; documented exception
-- (ADR 0011, issue #264) pending #267.
WITH closed AS (
    UPDATE jobs SET closed_at = NOW()
    WHERE primary_board_id = $1 AND closed_at IS NULL
      AND id IN (SELECT job_id FROM board_job_observations WHERE board_id = $1 AND last_snapshot_version < $2)
    RETURNING id
)
DELETE FROM option_answers WHERE job_id IN (SELECT id FROM closed);

-- name: CompletePollState :execrows
UPDATE board_poll_state
SET last_completed_at = NOW(),
    last_scheduled_at = CASE WHEN sqlc.arg(manual)::boolean THEN last_scheduled_at ELSE NOW() END,
    next_due_at = CASE WHEN sqlc.arg(manual)::boolean THEN next_due_at ELSE NOW() + sqlc.arg(interval_minutes)::int * INTERVAL '1 minute' END,
    consecutive_complete_empty = CASE WHEN sqlc.arg(empty)::boolean THEN consecutive_complete_empty + 1 ELSE 0 END,
    consecutive_failures = 0, lease_owner = NULL, lease_until = NULL
WHERE board_id = sqlc.arg(board_id)::uuid AND lease_owner = sqlc.arg(lease_owner)::text
  AND last_snapshot_version = sqlc.arg(version)::bigint;

-- name: RetireSupersededBoard :exec
UPDATE company_boards SET status = 'retired', retired_at = NOW()
WHERE id = $1 AND superseded_at IS NOT NULL AND status = 'verified';
