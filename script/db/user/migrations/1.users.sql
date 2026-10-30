-- +migrate Up
-- SQL in section 'Up' is executed when this migration is applied

CREATE TABLE users (
    id            INTEGER     PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    name          TEXT        NOT NULL UNIQUE,
    password_hash BYTEA       NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL
);

-- +migrate Down
-- SQL section 'Down' is executed when this migration is rolled back

DROP TABLE users;
