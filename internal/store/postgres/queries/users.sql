-- name: InsertUser :one
INSERT INTO users (username)
VALUES ($1)
ON CONFLICT (username) DO NOTHING
RETURNING id, username, created_at;

-- name: GetUserByUsername :one
SELECT id, username, created_at
FROM users
WHERE username = $1;

-- name: GetUserByID :one
SELECT id, username, created_at
FROM users
WHERE id = $1;

-- name: GetUsersByIDs :many
SELECT id, username, created_at
FROM users
WHERE id = ANY($1::bigint[]);
