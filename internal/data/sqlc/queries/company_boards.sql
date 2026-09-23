-- name: ListCompanyBoards :many
SELECT * FROM company_boards WHERE company_id = $1 ORDER BY created_at, id;

-- name: UpsertCandidateBoard :one
INSERT INTO company_boards (company_id, source, board_token)
VALUES ($1, $2, $3)
ON CONFLICT (source, board_token) DO UPDATE SET source = EXCLUDED.source
WHERE company_boards.company_id = EXCLUDED.company_id
RETURNING *;

-- name: VerifyCompanyBoard :one
UPDATE company_boards
SET status = 'verified', verification_method = $4, verified_at = NOW()
WHERE company_id = $1 AND source = $2 AND board_token = $3 AND status = 'candidate'
RETURNING *;
