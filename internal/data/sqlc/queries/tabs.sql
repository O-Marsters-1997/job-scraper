-- name: EnsureTabs :exec
INSERT INTO tracked_doc_tabs (tracked_doc_id, tab_id, title)
SELECT $1, unnest($2::text[]), unnest($3::text[])
ON CONFLICT (tracked_doc_id, tab_id) DO UPDATE SET title = EXCLUDED.title;

-- name: ListTabs :many
SELECT id, tracked_doc_id, tab_id, title, visible, created_at
FROM tracked_doc_tabs
WHERE tracked_doc_id = $1
ORDER BY created_at;

-- name: HideTab :execrows
UPDATE tracked_doc_tabs t
SET visible = FALSE
FROM tracked_docs d
WHERE t.tracked_doc_id = d.id
  AND d.user_id = $1 AND d.doc_id = $2 AND t.tab_id = $3;

-- name: ShowTab :execrows
UPDATE tracked_doc_tabs t
SET visible = TRUE
FROM tracked_docs d
WHERE t.tracked_doc_id = d.id
  AND d.user_id = $1 AND d.doc_id = $2 AND t.tab_id = $3;
