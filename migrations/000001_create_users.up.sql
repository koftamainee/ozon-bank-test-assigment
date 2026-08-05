CREATE TABLE users (
    id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    username   text NOT NULL UNIQUE,
    created_at timestamptz NOT NULL DEFAULT now()
);
