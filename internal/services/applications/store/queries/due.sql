-- name: ListDue :many
SELECT
    j.id           AS job_id,
    j.title        AS job_title,
    j.company_slug AS company_slug,
    COALESCE(co.kind, 'unclassified')::text AS company_kind,
    COALESCE(js.band, '')::text AS band,
    js.suitability_score AS score,
    a.id           AS application_id,
    a.chase_by     AS chase_by,
    ap.id          AS apply_task_id,
    COALESCE(ot.tasks, '[]'::jsonb)::jsonb AS tasks,
    d.id           AS draft_id,
    COALESCE(d.state, '')::text    AS draft_state,
    COALESCE(d.cv_title, '')::text AS draft_cv_title
FROM applications a
JOIN jobs j ON j.id = a.job_id
LEFT JOIN companies co ON co.id = j.company_id
LEFT JOIN job_scores js ON js.job_id = a.job_id AND js.user_id = a.user_id
LEFT JOIN LATERAL (
    SELECT t.id
    FROM tasks t JOIN task_types tt ON tt.id = t.type_id
    WHERE t.user_id = a.user_id AND t.job_id = a.job_id
      AND t.completed_at IS NULL AND tt.system_key = 'apply'
    ORDER BY t.created_at
    LIMIT 1
) ap ON TRUE
LEFT JOIN LATERAL (
    SELECT jsonb_agg(jsonb_build_object('id', t.id, 'type_name', tt.name, 'system_key', COALESCE(tt.system_key, ''), 'notes', COALESCE(t.notes, ''))
                     ORDER BY t.created_at, t.id) AS tasks
    FROM tasks t JOIN task_types tt ON tt.id = t.type_id
    WHERE t.user_id = a.user_id AND t.job_id = a.job_id
      AND t.completed_at IS NULL AND tt.system_key IS DISTINCT FROM 'apply'
) ot ON TRUE
LEFT JOIN LATERAL (
    SELECT c.id,
           CASE c.status
               WHEN 'failed' THEN 'failed'
               WHEN 'ready' THEN 'ready'
               WHEN 'keeping' THEN 'ready'
               ELSE 'pending'
           END AS state,
           COALESCE(tb.title, '') AS cv_title
    FROM tailored_cvs c
    LEFT JOIN tracked_docs td ON td.user_id = c.user_id AND td.doc_id = c.base_doc_id
    LEFT JOIN tracked_doc_tabs tb ON tb.tracked_doc_id = td.id AND tb.tab_id = c.base_tab_id
    WHERE c.user_id = a.user_id AND c.job_id = a.job_id
      AND c.outcome IS NULL AND ap.id IS NOT NULL
    ORDER BY c.created_at DESC
    LIMIT 1
) d ON TRUE
WHERE a.user_id = $1
  AND (a.chase_by IS NOT NULL OR ap.id IS NOT NULL OR ot.tasks IS NOT NULL OR d.id IS NOT NULL)
ORDER BY a.chase_by ASC NULLS LAST, j.title, a.id
LIMIT 500;
