-- name: OutreachSubject :one
SELECT
    COALESCE(c.name, '')::text AS name, c.role,
    COALESCE(co.name, c.company_name)::text AS company,
    COALESCE(ro.job_id::text, c.job_id::text, '')::text AS job_id,
    COALESCE(j.title, '')::text AS job_title,
    (SELECT COUNT(*) FROM contact_events s
     WHERE s.contact_id = c.id AND s.kind = 'sent'
       AND NOT EXISTS (
           SELECT 1 FROM contact_events r
           WHERE r.contact_id = c.id AND r.kind = 'reply'
             AND (r.occurred_on, r.created_at) > (s.occurred_on, s.created_at)))::int AS sent_since_reply,
    (ro.id IS NOT NULL AND ro.completed_at IS NULL)::bool AS reaching_out
FROM contacts c
LEFT JOIN companies co ON co.id = c.company_id
LEFT JOIN LATERAL (
    SELECT t.id, t.job_id, t.completed_at
    FROM tasks t
    JOIN task_types tt ON tt.id = t.type_id AND tt.system_key = 'reach_out'
    WHERE t.contact_id = c.id AND t.user_id = c.user_id
    ORDER BY (t.completed_at IS NULL) DESC, t.created_at DESC, t.id
    LIMIT 1
) ro ON true
LEFT JOIN jobs j ON j.id = COALESCE(ro.job_id, c.job_id)
WHERE c.id = $1 AND c.user_id = $2;

-- name: ListAchievementsNewestFirst :many
SELECT id, position_id, text, context FROM achievements
WHERE user_id = $1
ORDER BY created_at DESC, id DESC
LIMIT 1000;

-- name: UpsertOutreachHook :one
INSERT INTO outreach_hooks (contact_id, text, model, cost)
SELECT c.id, sqlc.arg(text)::text, sqlc.arg(model)::text, sqlc.arg(cost)::real
FROM contacts c
WHERE c.id = sqlc.arg(contact_id)::uuid AND c.user_id = sqlc.arg(user_id)::uuid
ON CONFLICT (contact_id) DO UPDATE
SET text = EXCLUDED.text, model = EXCLUDED.model,
    cost = outreach_hooks.cost + EXCLUDED.cost, created_at = NOW()
RETURNING contact_id;

-- name: GetOutreachHook :one
SELECT h.text, h.cost
FROM outreach_hooks h
JOIN contacts c ON c.id = h.contact_id
WHERE h.contact_id = $1 AND c.user_id = $2;
