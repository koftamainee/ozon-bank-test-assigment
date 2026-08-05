-- name: ListPosts :many
SELECT id, author_id, title, body, comments_allowed, deleted_at, created_at, updated_at
FROM posts
WHERE deleted_at IS NULL
ORDER BY created_at DESC, id DESC
LIMIT $1;

-- name: ListPostsAfter :many
SELECT id, author_id, title, body, comments_allowed, deleted_at, created_at, updated_at
FROM posts
WHERE deleted_at IS NULL
  AND (created_at < $1 OR (created_at = $1 AND id < $2))
ORDER BY created_at DESC, id DESC
LIMIT $3;

-- name: InsertPost :one
INSERT INTO posts (author_id, title, body)
VALUES ($1, $2, $3)
RETURNING id, author_id, title, body, comments_allowed, deleted_at, created_at, updated_at;

-- name: GetPostByID :one
SELECT id, author_id, title, body, comments_allowed, deleted_at, created_at, updated_at
FROM posts
WHERE id = $1;

-- name: UpdatePostCommentsAllowed :one
UPDATE posts
SET comments_allowed = $3, updated_at = now()
WHERE id = $1 AND author_id = $2 AND deleted_at IS NULL
RETURNING id, author_id, title, body, comments_allowed, deleted_at, created_at, updated_at;
