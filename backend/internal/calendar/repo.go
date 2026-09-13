package calendar

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/kabanos/backend/internal/postgres"
)

type Repo struct{ db *postgres.DB }

func NewRepo(db *postgres.DB) *Repo { return &Repo{db: db} }

const connCols = `id, user_id, provider, login, password_enc, calendar_url, calendar_name,
	enabled, last_sync_at, last_error, created_at, updated_at`

func scanConn(row pgx.Row) (*Connection, error) {
	var c Connection
	if err := row.Scan(&c.ID, &c.UserID, &c.Provider, &c.Login, &c.PasswordEnc, &c.CalendarURL,
		&c.CalendarName, &c.Enabled, &c.LastSyncAt, &c.LastError, &c.CreatedAt, &c.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, postgres.ErrNotFound
		}
		return nil, err
	}
	return &c, nil
}

func (r *Repo) GetConnection(ctx context.Context, userID uuid.UUID) (*Connection, error) {
	return scanConn(r.db.Read().QueryRow(ctx, `SELECT `+connCols+` FROM calendar_connections WHERE user_id=$1`, userID))
}

// Upsert creates or replaces the user's connection credentials + chosen calendar.
func (r *Repo) Upsert(ctx context.Context, userID uuid.UUID, provider, login string, passwordEnc []byte, calURL, calName string) (*Connection, error) {
	const q = `
		INSERT INTO calendar_connections (user_id, provider, login, password_enc, calendar_url, calendar_name, enabled)
		VALUES ($1,$2,$3,$4,$5,$6,true)
		ON CONFLICT (user_id) DO UPDATE SET
			provider=$2, login=$3, password_enc=$4, calendar_url=$5, calendar_name=$6,
			enabled=true, last_error='', updated_at=now()
		RETURNING ` + connCols
	return scanConn(r.db.Pool.QueryRow(ctx, q, userID, provider, login, passwordEnc, calURL, calName))
}

func (r *Repo) SetCalendar(ctx context.Context, userID uuid.UUID, url, name string) error {
	_, err := r.db.Pool.Exec(ctx,
		`UPDATE calendar_connections SET calendar_url=$2, calendar_name=$3, updated_at=now() WHERE user_id=$1`,
		userID, url, name)
	return err
}

func (r *Repo) SetSyncMeta(ctx context.Context, userID uuid.UUID, at time.Time, errStr string) error {
	_, err := r.db.Pool.Exec(ctx,
		`UPDATE calendar_connections SET last_sync_at=$2, last_error=$3, updated_at=now() WHERE user_id=$1`,
		userID, at, errStr)
	return err
}

func (r *Repo) Delete(ctx context.Context, userID uuid.UUID) error {
	ct, err := r.db.Pool.Exec(ctx, `DELETE FROM calendar_connections WHERE user_id=$1`, userID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return postgres.ErrNotFound
	}
	return nil
}

func (r *Repo) ListEnabled(ctx context.Context) ([]Connection, error) {
	rows, err := r.db.Read().Query(ctx, `SELECT `+connCols+` FROM calendar_connections WHERE enabled AND calendar_url<>''`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Connection{}
	for rows.Next() {
		c, err := scanConn(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *c)
	}
	return out, rows.Err()
}

// ---- items to push (local → remote) ----

// ItemsToPush returns calendar items that are new (no external_uid) or edited
// since their last sync (external_synced_at cleared by the dirty trigger).
func (r *Repo) ItemsToPush(ctx context.Context, userID uuid.UUID) ([]pushItem, error) {
	const q = `
		SELECT id, title, notes, scheduled_at, end_at, all_day, external_uid, external_href
		FROM gtd_items
		WHERE user_id=$1 AND bucket='calendar' AND NOT done AND scheduled_at IS NOT NULL
		  AND (external_uid IS NULL OR external_synced_at IS NULL)`
	rows, err := r.db.Read().Query(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []pushItem{}
	for rows.Next() {
		var p pushItem
		var sched time.Time
		if err := rows.Scan(&p.ID, &p.Title, &p.Notes, &sched, &p.EndAt, &p.AllDay, &p.ExternalUID, &p.ExternalHref); err != nil {
			return nil, err
		}
		p.ScheduledAt = sched
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *Repo) MarkItemSynced(ctx context.Context, userID, itemID uuid.UUID, uid, href, etag string) error {
	_, err := r.db.Pool.Exec(ctx,
		`UPDATE gtd_items SET external_uid=$3, external_href=$4, external_etag=$5, external_synced_at=now()
		 WHERE id=$1 AND user_id=$2`,
		itemID, userID, uid, href, etag)
	return err
}

// ---- pulled events (remote → local) ----

// UpsertPulledEvent mirrors a remote VEVENT into a calendar item, keyed by UID.
func (r *Repo) UpsertPulledEvent(ctx context.Context, userID uuid.UUID, ev remoteEvent) error {
	const q = `
		INSERT INTO gtd_items (user_id, title, notes, bucket, scheduled_at, end_at, all_day,
			external_uid, external_href, external_etag, external_synced_at)
		VALUES ($1,$2,$3,'calendar',$4,$5,$6,$7,$8,$9, now())
		ON CONFLICT (user_id, external_uid) WHERE external_uid IS NOT NULL DO UPDATE SET
			title=EXCLUDED.title, notes=EXCLUDED.notes, scheduled_at=EXCLUDED.scheduled_at,
			end_at=EXCLUDED.end_at, all_day=EXCLUDED.all_day, external_href=EXCLUDED.external_href,
			external_etag=EXCLUDED.external_etag, external_synced_at=now()`
	_, err := r.db.Pool.Exec(ctx, q, userID, ev.Summary, ev.Notes, ev.Start, ev.End, ev.AllDay,
		ev.UID, ev.Href, ev.ETag)
	return err
}

// ReconcileDeletions removes local mirror items whose UID no longer exists on
// the server within the synced window. It nulls external_href first so the
// delete trigger does not create a tombstone for an already-gone remote event.
func (r *Repo) ReconcileDeletions(ctx context.Context, userID uuid.UUID, keepUIDs []string, from, to time.Time) (int, error) {
	tx, err := r.db.Pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)

	const cond = `user_id=$1 AND bucket='calendar' AND external_uid IS NOT NULL
		AND scheduled_at >= $2 AND scheduled_at < $3 AND NOT (external_uid = ANY($4))`
	if _, err := tx.Exec(ctx, `UPDATE gtd_items SET external_href=NULL WHERE `+cond, userID, from, to, keepUIDs); err != nil {
		return 0, err
	}
	ct, err := tx.Exec(ctx, `DELETE FROM gtd_items WHERE `+cond, userID, from, to, keepUIDs)
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return int(ct.RowsAffected()), nil
}

// ---- tombstones ----

func (r *Repo) ListTombstones(ctx context.Context, userID uuid.UUID) ([]tombstone, error) {
	rows, err := r.db.Read().Query(ctx,
		`SELECT id, external_href, external_etag FROM calendar_tombstones WHERE user_id=$1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []tombstone{}
	for rows.Next() {
		var t tombstone
		if err := rows.Scan(&t.ID, &t.Href, &t.ETag); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (r *Repo) DeleteTombstone(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Pool.Exec(ctx, `DELETE FROM calendar_tombstones WHERE id=$1`, id)
	return err
}
