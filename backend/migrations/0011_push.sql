-- +goose Up
-- Web Push subscriptions: one row per device/browser the user has enabled
-- notifications on. Keys are the client-generated P-256 ECDH public key and
-- auth secret required to encrypt push payloads for that endpoint.

CREATE TABLE push_subscriptions (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    endpoint   TEXT NOT NULL UNIQUE,
    p256dh     TEXT NOT NULL,
    auth       TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX push_subscriptions_user_idx ON push_subscriptions (user_id);

-- +goose Down
DROP TABLE push_subscriptions;
