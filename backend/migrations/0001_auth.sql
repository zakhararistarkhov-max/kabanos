-- +goose Up
-- Core identity + the token tables that back email verification, password
-- reset and refresh-token rotation. UUID primary keys keep ids opaque and
-- non-enumerable; timestamptz everywhere avoids timezone ambiguity.

CREATE TABLE users (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email            TEXT        NOT NULL,
    password_hash    TEXT        NOT NULL,
    display_name     TEXT        NOT NULL DEFAULT '',
    -- Profile fields used by other domains (BMI, calorie targets).
    height_cm        NUMERIC(5,2),
    sex              TEXT CHECK (sex IN ('male','female','other')),
    birth_date       DATE,
    -- Telegram handle for the morning digest bot (linked later).
    telegram_username TEXT,
    telegram_chat_id  BIGINT,
    email_verified_at TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Case-insensitive uniqueness on email without requiring the citext extension.
CREATE UNIQUE INDEX users_email_lower_key ON users (lower(email));

CREATE TABLE refresh_tokens (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    -- Only the hash is stored; the raw token lives solely in the client cookie.
    token_hash   TEXT NOT NULL UNIQUE,
    expires_at   TIMESTAMPTZ NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    revoked_at   TIMESTAMPTZ,
    -- Rotation chain: when a token is used, it is revoked and points at its
    -- successor. Reuse of a revoked token signals theft → revoke the family.
    replaced_by  UUID REFERENCES refresh_tokens(id) ON DELETE SET NULL,
    user_agent   TEXT NOT NULL DEFAULT '',
    ip           TEXT NOT NULL DEFAULT ''
);
CREATE INDEX refresh_tokens_user_id_idx ON refresh_tokens (user_id);

CREATE TABLE email_verification_tokens (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash  TEXT NOT NULL UNIQUE,
    expires_at  TIMESTAMPTZ NOT NULL,
    consumed_at TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX email_verification_tokens_user_id_idx ON email_verification_tokens (user_id);

CREATE TABLE password_reset_tokens (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash  TEXT NOT NULL UNIQUE,
    expires_at  TIMESTAMPTZ NOT NULL,
    consumed_at TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX password_reset_tokens_user_id_idx ON password_reset_tokens (user_id);

-- Transactional outbox: domain events (send email, push telegram digest) are
-- written in the same DB transaction as the business change, then delivered
-- asynchronously by the worker. This gives at-least-once delivery without a
-- separate message broker and survives crashes.
CREATE TABLE outbox_messages (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    topic        TEXT NOT NULL,
    payload      JSONB NOT NULL,
    status       TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','processing','done','failed')),
    attempts     INT  NOT NULL DEFAULT 0,
    last_error   TEXT NOT NULL DEFAULT '',
    available_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    processed_at TIMESTAMPTZ
);
-- Worker claim query filters by status + availability; index accordingly.
CREATE INDEX outbox_pending_idx ON outbox_messages (available_at)
    WHERE status IN ('pending','failed');

-- +goose Down
DROP TABLE outbox_messages;
DROP TABLE password_reset_tokens;
DROP TABLE email_verification_tokens;
DROP TABLE refresh_tokens;
DROP INDEX IF EXISTS users_email_lower_key;
DROP TABLE users;
