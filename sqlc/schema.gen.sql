-- Code generated from migrations/*.up.sql by 'make sqlc'. DO NOT EDIT.
CREATE TABLE users (
    id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    username   text NOT NULL UNIQUE,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE posts (
    id               bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    author_id        bigint NOT NULL REFERENCES users(id),
    title            text NOT NULL,
    body             text NOT NULL,
    comments_allowed boolean NOT NULL DEFAULT true,
    deleted_at       timestamptz,
    created_at       timestamptz NOT NULL DEFAULT now(),
    updated_at       timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_posts_created_at ON posts (created_at DESC, id DESC);
CREATE TABLE comments (
    id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    post_id    bigint NOT NULL REFERENCES posts(id),
    author_id  bigint NOT NULL REFERENCES users(id),
    parent_id  bigint REFERENCES comments(id),
    path       text NOT NULL,
    body       text NOT NULL CHECK (char_length(body) BETWEEN 1 AND 2000),
    deleted_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_comments_post_path ON comments (post_id, path);
CREATE INDEX idx_comments_parent ON comments (parent_id);
