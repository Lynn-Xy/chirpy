-- name: CreateUser :one
INSERT INTO users (id, created_at, updated_at, email, hashed_password, is_chirpy_red)
VALUES (gen_random_uuid(), NOW(), NOW(), $1, $2, FALSE)
RETURNING *;

-- name: UpdateUserEmail :one
UPDATE users
SET email = $2, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: GetUserByEmail :one
SELECT id, created_at, updated_at, email, hashed_password, is_chirpy_red
FROM users
WHERE email = $1;

-- name: UpgradeUserToChirpyRed :one
UPDATE users
SET is_chirpy_red = TRUE, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteAllUsers :exec
DELETE FROM users;