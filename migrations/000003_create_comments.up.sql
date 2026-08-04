CREATE TABLE comments (
    id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    post_id    bigint NOT NULL REFERENCES posts(id),
    author_id  bigint NOT NULL REFERENCES users(id),
    parent_id  bigint REFERENCES comments(id),
    path       text NOT NULL,
    depth      smallint NOT NULL DEFAULT 0,
    body       text NOT NULL CHECK (char_length(body) BETWEEN 1 AND 2000),
    deleted_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_comments_post_path ON comments (post_id, path);
CREATE INDEX idx_comments_parent ON comments (parent_id);
