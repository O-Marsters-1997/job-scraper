-- name: GetUserProfile :one
SELECT id, username, email, cv_filename_pattern FROM users WHERE id = $1;

-- name: UpdateUserEmail :one
UPDATE users SET email = $2 WHERE id = $1 RETURNING id, username, email, cv_filename_pattern;

-- name: SetUserCVFilenamePattern :execrows
UPDATE users SET cv_filename_pattern = $2 WHERE id = $1;
