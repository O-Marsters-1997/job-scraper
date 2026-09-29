-- name: ListHeadingMappings :many
SELECT * FROM cv_heading_mappings
WHERE user_id = $1 AND doc_id = $2 AND tab_id = $3
ORDER BY heading_text;

-- name: UpsertHeadingMapping :exec
INSERT INTO cv_heading_mappings (user_id, doc_id, tab_id, heading_text, position_id)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (user_id, doc_id, tab_id, heading_text)
DO UPDATE SET position_id = EXCLUDED.position_id, updated_at = NOW();
