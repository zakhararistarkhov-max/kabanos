-- +goose Up
-- Personal diary: free-text entries with photo/video attachments, kept per
-- calendar day. Attachments are stored as a JSONB array of object references
-- ({key, kind, contentType}); the media files themselves live in S3/MinIO and
-- are served to the browser via short-lived presigned URLs.

CREATE TABLE diary_entries (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    entry_date  DATE NOT NULL,
    body        TEXT NOT NULL DEFAULT '',
    attachments JSONB NOT NULL DEFAULT '[]',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX diary_entries_user_date_idx ON diary_entries (user_id, entry_date, created_at);

-- +goose Down
DROP TABLE diary_entries;
