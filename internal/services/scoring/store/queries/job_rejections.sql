-- name: UpsertJobRejection :exec
INSERT INTO job_rejections (user_id, job_id, kind, reason, search_config_version, content_fingerprint, rejected_at)
VALUES ($1, $2, $3, $4, $5, $6, sqlc.arg(rejected_at)::timestamptz)
ON CONFLICT (user_id, job_id) DO UPDATE SET
    kind                  = EXCLUDED.kind,
    reason                = EXCLUDED.reason,
    search_config_version = EXCLUDED.search_config_version,
    content_fingerprint   = EXCLUDED.content_fingerprint,
    rejected_at           = EXCLUDED.rejected_at;

-- name: DeleteJobRejection :exec
DELETE FROM job_rejections WHERE user_id = $1 AND job_id = $2;

-- name: ListJobRejections :many
SELECT r.job_id, r.kind, r.reason, r.search_config_version, r.rejected_at,
    (r.content_fingerprint IS NOT DISTINCT FROM j.content_fingerprint
        AND r.search_config_version IS NOT DISTINCT FROM sc.updated_at)::boolean AS is_current
FROM job_rejections r
JOIN jobs j ON j.id = r.job_id
LEFT JOIN search_config sc ON sc.user_id = r.user_id
WHERE r.user_id = sqlc.arg(user_id)::uuid
    AND (sqlc.narg(before_rejected_at)::timestamptz IS NULL
        OR (r.rejected_at, r.job_id) < (sqlc.narg(before_rejected_at)::timestamptz, sqlc.narg(before_job_id)::uuid))
ORDER BY r.rejected_at DESC, r.job_id DESC
LIMIT sqlc.arg(row_limit)::int;
