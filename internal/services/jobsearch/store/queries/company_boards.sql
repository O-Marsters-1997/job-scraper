-- name: ListCompanyBoards :many
SELECT * FROM company_boards WHERE company_id = $1 ORDER BY created_at, id;

-- name: UpsertCandidateBoard :one
INSERT INTO company_boards (company_id, source, board_token)
VALUES ($1, $2, $3)
ON CONFLICT (source, board_token) DO UPDATE SET source = EXCLUDED.source
WHERE company_boards.company_id = EXCLUDED.company_id
RETURNING *;

-- name: GetVerifiedBoardID :one
SELECT id FROM company_boards WHERE source = $1 AND board_token = $2 AND status = 'verified';

-- name: GetBoardCompanyID :one
SELECT company_id FROM company_boards WHERE source = $1 AND board_token = $2;

-- name: ListBoardChecks :many
SELECT b.id, s.last_completed_at FROM company_boards b
LEFT JOIN board_poll_state s ON s.board_id = b.id
WHERE b.company_id = $1;

-- name: VerifyCompanyBoard :one
UPDATE company_boards
SET status = 'verified', verification_method = $4, verified_at = NOW()
WHERE company_id = $1 AND source = $2 AND board_token = $3 AND status = 'candidate'
RETURNING *;

-- name: ListTrackedCompanyBoards :many
SELECT cb.id, cb.company_id, cb.source, cb.board_token, cb.status
FROM company_boards cb
JOIN tracked_companies tc ON tc.company_id = cb.company_id
WHERE tc.user_id = $1
ORDER BY cb.created_at, cb.id;

-- name: ListVerifiedBoardsBySlug :many
SELECT c.slug, cb.source, cb.board_token,
  EXISTS (SELECT 1 FROM tracked_companies tc WHERE tc.company_id = c.id AND tc.enabled) AS tracked
FROM companies c
JOIN company_boards cb ON cb.company_id = c.id AND cb.status = 'verified'
WHERE c.slug = ANY($1::text[])
ORDER BY c.slug, cb.created_at, cb.id;

-- name: ListVerifiedCompanySlugs :many
SELECT DISTINCT c.slug
FROM companies c
JOIN company_boards cb ON cb.company_id = c.id AND cb.status = 'verified'
WHERE c.slug = ANY($1::text[]);

-- name: ListUntrackedDiscoveredBoards :many
SELECT cb.* FROM company_boards cb
WHERE cb.status = 'verified' AND cb.verification_method = 'discovered'
  AND NOT EXISTS (SELECT 1 FROM tracked_companies tc WHERE tc.company_id = cb.company_id)
ORDER BY cb.created_at, cb.id;
