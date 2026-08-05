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
