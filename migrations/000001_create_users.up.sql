-- NOTE: i am using single JWT token to identify user, replace for refresh and
-- NOTE: access token in production
CREATE TABLE users (
    id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    username   text NOT NULL UNIQUE,
    created_at timestamptz NOT NULL DEFAULT now()
);
