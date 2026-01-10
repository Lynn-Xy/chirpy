-- name: UpdatePasswordHash :exec
UPDATE users
SET hashed_password = $2, updated_at = NOW()
WHERE id = $1;

-- name: GetPasswordHash :one
SELECT hashed_password
FROM users
WHERE id = $1;