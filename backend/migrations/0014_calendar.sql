-- +goose Up
-- Two-way calendar sync (Yandex.Calendar over CalDAV). A per-user connection
-- holds the CalDAV credentials (app password encrypted at rest) and the chosen
-- calendar collection. GTD calendar items (bucket='calendar') gain external_*
-- columns linking them to their remote VEVENT so changes flow both ways.

CREATE TABLE calendar_connections (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    provider      TEXT NOT NULL DEFAULT 'yandex',
    login         TEXT NOT NULL,
    password_enc  BYTEA NOT NULL,           -- AES-GCM(nonce || ciphertext)
    calendar_url  TEXT NOT NULL DEFAULT '', -- chosen calendar collection path
    calendar_name TEXT NOT NULL DEFAULT '',
    enabled       BOOLEAN NOT NULL DEFAULT true,
    last_sync_at  TIMESTAMPTZ,
    last_error    TEXT NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE gtd_items
    ADD COLUMN external_uid       TEXT,
    ADD COLUMN external_href      TEXT,
    ADD COLUMN external_etag      TEXT,
    ADD COLUMN external_synced_at TIMESTAMPTZ;

-- One local row per remote event (per user).
CREATE UNIQUE INDEX gtd_items_ext_uid_idx ON gtd_items (user_id, external_uid)
    WHERE external_uid IS NOT NULL;

-- Deletions of synced calendar items are recorded so the next sync can remove
-- the matching event on the server (the row itself is already gone by then).
CREATE TABLE calendar_tombstones (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    external_href TEXT NOT NULL,
    external_etag TEXT NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX calendar_tombstones_user_idx ON calendar_tombstones (user_id);

-- +goose StatementBegin
-- On delete of a synced calendar item, leave a tombstone for the sync worker.
CREATE FUNCTION gtd_items_tombstone() RETURNS trigger AS $$
BEGIN
    IF OLD.external_href IS NOT NULL AND OLD.external_href <> '' THEN
        INSERT INTO calendar_tombstones (user_id, external_href, external_etag)
        VALUES (OLD.user_id, OLD.external_href, COALESCE(OLD.external_etag, ''));
    END IF;
    RETURN OLD;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER gtd_items_tombstone_trg
    BEFORE DELETE ON gtd_items
    FOR EACH ROW EXECUTE FUNCTION gtd_items_tombstone();

-- +goose StatementBegin
-- When the app edits a synced calendar item's content, clear external_synced_at
-- so the next sync pushes the change. The sync itself sets external_synced_at
-- explicitly, which this guard detects (NEW differs from OLD) and leaves alone —
-- so pulled updates don't get re-marked dirty.
CREATE FUNCTION gtd_items_mark_dirty() RETURNS trigger AS $$
BEGIN
    IF NEW.bucket = 'calendar'
       AND NEW.external_synced_at IS NOT DISTINCT FROM OLD.external_synced_at
       AND (NEW.title       IS DISTINCT FROM OLD.title
         OR NEW.notes       IS DISTINCT FROM OLD.notes
         OR NEW.scheduled_at IS DISTINCT FROM OLD.scheduled_at
         OR NEW.end_at      IS DISTINCT FROM OLD.end_at
         OR NEW.all_day     IS DISTINCT FROM OLD.all_day) THEN
        NEW.external_synced_at := NULL;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER gtd_items_mark_dirty_trg
    BEFORE UPDATE ON gtd_items
    FOR EACH ROW EXECUTE FUNCTION gtd_items_mark_dirty();

-- +goose Down
DROP TRIGGER IF EXISTS gtd_items_mark_dirty_trg ON gtd_items;
DROP FUNCTION IF EXISTS gtd_items_mark_dirty();
DROP TRIGGER IF EXISTS gtd_items_tombstone_trg ON gtd_items;
DROP FUNCTION IF EXISTS gtd_items_tombstone();
DROP TABLE calendar_tombstones;
DROP INDEX IF EXISTS gtd_items_ext_uid_idx;
ALTER TABLE gtd_items
    DROP COLUMN external_uid,
    DROP COLUMN external_href,
    DROP COLUMN external_etag,
    DROP COLUMN external_synced_at;
DROP TABLE calendar_connections;
