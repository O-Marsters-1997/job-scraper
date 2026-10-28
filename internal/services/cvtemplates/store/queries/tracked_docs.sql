-- name: AddTrackedDoc :exec
INSERT INTO tracked_docs (user_id, doc_id, kind)
VALUES ($1, $2, $3)
ON CONFLICT DO NOTHING;

-- name: RemoveTrackedDoc :execrows
DELETE FROM tracked_docs WHERE user_id = $1 AND doc_id = $2;

-- name: ListTrackedDocs :many
SELECT id, user_id, doc_id, kind, added_at
FROM tracked_docs
WHERE user_id = $1 AND kind = $2
ORDER BY added_at;
