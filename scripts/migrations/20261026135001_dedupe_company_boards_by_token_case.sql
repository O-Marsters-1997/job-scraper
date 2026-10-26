-- +goose Up
CREATE TEMP TABLE board_merge AS
WITH ranked AS (
    SELECT b.id, b.source, lower(b.board_token) AS token_key,
        row_number() OVER (
            PARTITION BY b.source, lower(b.board_token)
            ORDER BY (SELECT count(*) FROM jobs j WHERE j.primary_board_id = b.id) DESC, b.created_at, b.id
        ) AS rank
    FROM company_boards b
    WHERE b.status <> 'retired'
)
SELECT l.id AS loser_id, w.id AS survivor_id
FROM ranked l
JOIN ranked w ON w.source = l.source AND w.token_key = l.token_key AND w.rank = 1
WHERE l.rank > 1;

UPDATE jobs SET primary_board_id = m.survivor_id
FROM board_merge m
WHERE jobs.primary_board_id = m.loser_id;

INSERT INTO board_job_observations (board_id, job_id, last_seen_at, last_snapshot_version)
SELECT m.survivor_id, o.job_id, o.last_seen_at, o.last_snapshot_version
FROM board_job_observations o
JOIN board_merge m ON m.loser_id = o.board_id
ON CONFLICT ON CONSTRAINT board_job_observations_pkey DO NOTHING;

DELETE FROM board_job_observations WHERE board_id IN (SELECT loser_id FROM board_merge);

UPDATE company_boards
SET status = 'retired', retired_at = NOW(), board_token = board_token || '#' || id
WHERE id IN (SELECT loser_id FROM board_merge);

DROP TABLE board_merge;

UPDATE company_boards SET board_token = lower(board_token)
WHERE status <> 'retired' AND board_token <> lower(board_token)
  AND NOT EXISTS (SELECT 1 FROM company_boards o WHERE o.source = company_boards.source AND o.board_token = lower(company_boards.board_token))
  AND source IN ('ashby', 'smartrecruiters', 'pinpoint', 'recruitee', 'personio', 'teamtailor', 'hibob');

CREATE UNIQUE INDEX company_boards_source_token_lower_key
    ON company_boards (source, lower(board_token)) WHERE status <> 'retired';

-- +goose Down
DROP INDEX company_boards_source_token_lower_key;
