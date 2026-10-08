-- name: ListBankSkills :many
SELECT * FROM bank_skills
WHERE user_id = $1
ORDER BY sort_order, created_at;

-- name: CreateBankSkill :one
INSERT INTO bank_skills (user_id, name, category, sort_order)
VALUES (
    sqlc.arg(user_id), sqlc.arg(name), sqlc.arg(category),
    (SELECT COALESCE(MAX(s.sort_order), 0) + 1 FROM bank_skills s WHERE s.user_id = sqlc.arg(user_id))
)
RETURNING *;

-- name: UpdateBankSkill :one
UPDATE bank_skills
SET name = sqlc.arg(name), category = sqlc.arg(category), updated_at = NOW()
WHERE user_id = sqlc.arg(user_id) AND id = sqlc.arg(id)
RETURNING *;

-- name: DeleteBankSkill :execrows
DELETE FROM bank_skills WHERE user_id = $1 AND id = $2;

-- name: ReorderBankSkills :execrows
UPDATE bank_skills s
SET sort_order = o.ord::int
FROM unnest(sqlc.arg(ids)::uuid[]) WITH ORDINALITY AS o(id, ord)
WHERE s.user_id = sqlc.arg(user_id) AND s.id = o.id;
