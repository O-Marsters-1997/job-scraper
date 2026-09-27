-- name: AddTrackedDoc :exec
INSERT INTO tracked_docs (user_id, doc_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: RemoveTrackedDoc :execrows
DELETE FROM tracked_docs WHERE user_id = $1 AND doc_id = $2;

-- name: ListTrackedDocs :many
SELECT id, user_id, doc_id, added_at
FROM tracked_docs
WHERE user_id = $1
ORDER BY added_at;
