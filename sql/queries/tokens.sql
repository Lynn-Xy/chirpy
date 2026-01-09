-- name: CreateRefreshToken :one
INSERT INTO refresh_tokens (token, created_at, updated_at, user_id, expires_at, revoked_at)
VALUES ($1, NOW(), NOW(), $2, $3, NULL)
RETURNING *;

-- name: GetUserFromRefreshToken :one
SELECT u.id, u.created_at, u.updated_at, u.email, u.hashed_password
FROM users u
JOIN refresh_tokens rt ON u.id = rt.user_id
WHERE rt.token = $1 AND (rt.revoked_at IS NULL AND rt.expires_at > NOW());

-- name: RevokeRefreshToken :one
UPDATE refresh_tokens
SET revoked_at = NOW(),
    updated_at = NOW()
WHERE token = $1
RETURNING *;

-- name: GetRefreshToken :one
SELECT token, created_at, updated_at, user_id, expires_at, revoked_at
FROM refresh_tokens
WHERE token = $1;