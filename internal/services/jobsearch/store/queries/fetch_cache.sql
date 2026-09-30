-- name: GetFetchCache :one
SELECT url, status, header, body FROM fetch_cache WHERE url = $1;

-- name: UpsertFetchCache :exec
INSERT INTO fetch_cache (url, status, header, body) VALUES ($1, $2, $3, $4)
ON CONFLICT (url) DO UPDATE SET status = EXCLUDED.status, header = EXCLUDED.header, body = EXCLUDED.body, fetched_at = NOW();

-- name: DeleteFetchCacheURLs :exec
DELETE FROM fetch_cache WHERE url = ANY(@urls::text[]);

-- name: DeleteExpiredFetchCache :exec
DELETE FROM fetch_cache WHERE fetched_at < NOW() - INTERVAL '7 days';
