-- name: ListOutreachTemplates :many
SELECT id, user_id, kind, name, body, is_default, created_at FROM outreach_templates
WHERE user_id = $1
ORDER BY kind, created_at, id
LIMIT 500;

-- name: GetOutreachTemplate :one
SELECT id, user_id, kind, name, body, is_default, created_at FROM outreach_templates
WHERE user_id = $1 AND id = $2
FOR UPDATE;

-- name: CreateOutreachTemplate :one
INSERT INTO outreach_templates (user_id, kind, name, body, is_default)
VALUES (
    sqlc.arg(user_id), sqlc.arg(kind), sqlc.arg(name), sqlc.arg(body),
    NOT EXISTS (SELECT 1 FROM outreach_templates o WHERE o.user_id = sqlc.arg(user_id) AND o.kind = sqlc.arg(kind))
)
RETURNING id, user_id, kind, name, body, is_default, created_at;

-- name: UpdateOutreachTemplate :one
UPDATE outreach_templates SET name = $3, body = $4
WHERE user_id = $1 AND id = $2
RETURNING id, user_id, kind, name, body, is_default, created_at;

-- name: ClearOutreachDefault :exec
UPDATE outreach_templates SET is_default = FALSE
WHERE user_id = $1 AND kind = $2 AND is_default;

-- name: SetOutreachDefault :one
UPDATE outreach_templates SET is_default = TRUE
WHERE user_id = $1 AND id = $2
RETURNING id, user_id, kind, name, body, is_default, created_at;

-- name: CountOutreachTemplatesOfKind :one
SELECT count(*) FROM outreach_templates WHERE user_id = $1 AND kind = $2;

-- name: DeleteOutreachTemplate :exec
DELETE FROM outreach_templates WHERE user_id = $1 AND id = $2;

-- name: SeedOutreachTemplate :exec
INSERT INTO outreach_templates (user_id, kind, name, body, is_default)
VALUES ($1, $2, $3, $4, TRUE);
