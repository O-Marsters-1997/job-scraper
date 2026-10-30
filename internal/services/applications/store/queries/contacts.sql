-- name: CreateContact :one
INSERT INTO contacts (user_id, name, role, company_id, company_name, job_id, linkedin_url, email)
SELECT sqlc.arg('user_id')::uuid, sqlc.arg('name'), sqlc.arg('role'),
       COALESCE(sqlc.narg('company_id')::uuid, j.company_id), sqlc.arg('company_name'),
       sqlc.narg('job_id')::uuid, sqlc.narg('linkedin_url'), sqlc.narg('email')
FROM (SELECT 1) one
LEFT JOIN jobs j ON j.id = sqlc.narg('job_id')::uuid
RETURNING id;

-- name: LockContact :one
SELECT stage FROM contacts WHERE id = $1 AND user_id = $2 FOR UPDATE;

-- name: CountSentSinceReply :one
SELECT COUNT(*)::int FROM contact_events s
WHERE s.contact_id = $1 AND s.kind = 'sent'
  AND NOT EXISTS (
      SELECT 1 FROM contact_events r
      WHERE r.contact_id = s.contact_id AND r.kind = 'reply'
        AND (r.occurred_on, r.created_at) > (s.occurred_on, s.created_at));

-- name: InsertContactEvent :exec
INSERT INTO contact_events (contact_id, kind, occurred_on, channel, template_kind, template_name, outcome, job_id)
SELECT c.id, sqlc.arg('kind'), sqlc.arg('occurred_on')::date, sqlc.narg('channel'), sqlc.narg('template_kind'),
       sqlc.arg('template_name'), sqlc.narg('outcome'), c.job_id
FROM contacts c WHERE c.id = sqlc.arg('contact_id');

-- name: SetContactState :exec
UPDATE contacts
SET stage = sqlc.arg('stage'), outcome = sqlc.narg('outcome'), follow_up_by = sqlc.narg('follow_up_by')::date,
    reply = COALESCE(sqlc.narg('reply'), reply), updated_at = NOW()
WHERE id = sqlc.arg('id') AND user_id = sqlc.arg('user_id');

-- name: SetLinkedApplicationChase :execrows
UPDATE applications a
SET chase_by = sqlc.narg('chase_by')::date
FROM contacts c
WHERE c.id = sqlc.arg('contact_id') AND c.user_id = sqlc.arg('user_id')
  AND a.user_id = c.user_id AND a.job_id = c.job_id;

-- name: ListContacts :many
SELECT
    c.id, c.name, c.role, c.company_id,
    COALESCE(co.name, c.company_name)::text AS company_name,
    COALESCE(co.kind, '')::text AS company_kind,
    c.job_id,
    COALESCE(j.title, '')::text AS job_title,
    a.id AS application_id,
    COALESCE(c.linkedin_url, '')::text AS linkedin_url,
    COALESCE(c.email, '')::text AS email,
    c.stage, c.outcome, c.follow_up_by,
    COALESCE(c.reply, '')::text AS reply,
    (SELECT COUNT(*) FROM contact_events s
     WHERE s.contact_id = c.id AND s.kind = 'sent'
       AND NOT EXISTS (
           SELECT 1 FROM contact_events r
           WHERE r.contact_id = c.id AND r.kind = 'reply'
             AND (r.occurred_on, r.created_at) > (s.occurred_on, s.created_at)))::int AS sent_since_reply,
    c.created_at
FROM contacts c
LEFT JOIN companies co ON co.id = c.company_id
LEFT JOIN jobs j ON j.id = c.job_id
LEFT JOIN applications a ON a.user_id = c.user_id AND a.job_id = c.job_id
WHERE c.user_id = sqlc.arg('user_id')
  AND (sqlc.narg('id')::uuid IS NULL OR c.id = sqlc.narg('id')::uuid)
ORDER BY c.created_at DESC, c.id
LIMIT 500;

-- name: ListContactEvents :many
SELECT e.id, e.contact_id, e.kind, e.occurred_on, e.channel, e.template_kind, e.template_name,
       e.outcome, e.note, e.job_id, COALESCE(j.title, '')::text AS job_title
FROM contact_events e
LEFT JOIN jobs j ON j.id = e.job_id
WHERE e.contact_id = ANY(sqlc.arg('contact_ids')::uuid[])
ORDER BY e.occurred_on, e.created_at, (e.kind = 'closed');

-- name: DeleteContact :execrows
DELETE FROM contacts WHERE id = $1 AND user_id = $2;
