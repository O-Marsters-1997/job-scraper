-- name: GetUserProfile :one
SELECT id, username, email FROM users WHERE id = $1;

-- name: UpdateUserEmail :one
UPDATE users SET email = $2 WHERE id = $1 RETURNING id, username, email;
