-- name: RecordDiscoverMiss :exec
INSERT INTO board_discover_misses (source, board_token) VALUES ($1, $2)
ON CONFLICT (source, board_token) DO UPDATE SET missed_at = NOW();

-- name: DiscoverMissedSince :one
SELECT EXISTS (
    SELECT 1 FROM board_discover_misses
    WHERE source = $1 AND board_token = $2 AND missed_at >= $3
);
