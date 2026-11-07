-- Auth tables (SDS §2.2, F-01 contract §4.1). IDs are UUIDv7, generated in Go.

-- +goose Up
CREATE TABLE "user" (
    id               UUID PRIMARY KEY,
    name             VARCHAR(100) NOT NULL,
    email            VARCHAR(255) NOT NULL UNIQUE,
    email_verified   BOOLEAN      NOT NULL DEFAULT false,
    image            TEXT,
    username         VARCHAR(50)  NOT NULL UNIQUE CHECK (username = lower(username)),
    display_username VARCHAR(50)  NOT NULL,
    role             VARCHAR(20)  NOT NULL DEFAULT 'requester'
                     CHECK (role IN ('requester', 'reviewer', 'finance')),
    banned           BOOLEAN      NOT NULL DEFAULT false,
    ban_reason       VARCHAR(200),
    ban_expires      TIMESTAMPTZ,
    position         VARCHAR(50)  NOT NULL,
    reviewer_id      UUID REFERENCES "user" (id) CHECK (reviewer_id <> id),
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE TABLE account (
    id          UUID PRIMARY KEY,
    user_id     UUID         NOT NULL REFERENCES "user" (id) ON DELETE CASCADE,
    provider_id VARCHAR(50)  NOT NULL,
    account_id  VARCHAR(255) NOT NULL,
    password    VARCHAR(100),
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    UNIQUE (provider_id, account_id),
    CHECK (provider_id <> 'credential' OR password IS NOT NULL)
);

CREATE INDEX account_user_id_idx ON account (user_id);

CREATE TABLE session (
    id         UUID PRIMARY KEY,
    user_id    UUID        NOT NULL REFERENCES "user" (id) ON DELETE CASCADE,
    token      BYTEA       NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    ip_address VARCHAR(45),
    user_agent TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX session_user_id_idx ON session (user_id);

-- +goose Down
DROP TABLE session;
DROP TABLE account;
DROP TABLE "user";
