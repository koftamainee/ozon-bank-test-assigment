-- name: ListCommentsByPost :many
SELECT id, post_id, author_id, parent_id, path, body, deleted_at, created_at
FROM comments
WHERE post_id = $1 AND deleted_at IS NULL
ORDER BY path ASC
LIMIT $2;

-- name: ListCommentsByPostAfter :many
SELECT id, post_id, author_id, parent_id, path, body, deleted_at, created_at
FROM comments
WHERE post_id = $1 AND deleted_at IS NULL AND path > $2
ORDER BY path ASC
LIMIT $3;

-- name: GetCommentByID :one
SELECT id, post_id, author_id, parent_id, path, body, deleted_at, created_at
FROM comments
WHERE id = $1;

-- name: LockPostForComments :one
SELECT comments_allowed, deleted_at
FROM posts
WHERE id = $1
FOR SHARE;

-- name: LockCommentForReply :one
SELECT post_id, path, deleted_at
FROM comments
WHERE id = $1
FOR SHARE;

-- name: InsertComment :one
INSERT INTO comments (post_id, author_id, parent_id, path, body)
VALUES ($1, $2, $3, '', $4)
RETURNING id, post_id, author_id, parent_id, path, body, deleted_at, created_at;

-- name: UpdateCommentPath :one
UPDATE comments
SET path = $1
WHERE id = $2
RETURNING id, post_id, author_id, parent_id, path, body, deleted_at, created_at;
