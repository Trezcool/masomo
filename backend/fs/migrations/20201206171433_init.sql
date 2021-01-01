-- +goose Up
-- SQL in this section is executed when the migration is applied.
CREATE TABLE "user" (
    id              UUID            NOT NULL,
    name            VARCHAR(100),
    username        VARCHAR(254)    UNIQUE,
    email           VARCHAR(254)    UNIQUE, -- 254: to be compliant with RFCs 3696 and 5321
    is_active       BOOL,
    roles           TEXT[],
    password_hash   BYTEA,
    created_at      TIMESTAMP,
    updated_at      TIMESTAMP,
    last_login      TIMESTAMP,

    PRIMARY KEY (id)
);

-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
DROP TABLE "user";
