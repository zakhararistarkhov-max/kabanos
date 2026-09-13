package calendar

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Sync window: how far back and forward remote events are mirrored.
const (
	windowBack    = 7 * 24 * time.Hour
	windowForward = 60 * 24 * time.Hour
)

type Service struct {
	repo  *Repo
	crypt cryptor
	log   *slog.Logger
}

func NewService(repo *Repo, encKey []byte, log *slog.Logger) (*Service, error) {
	cr, err := newCryptor(encKey)
	if err != nil {
		return nil, err
	}
	return &Service{repo: repo, crypt: cr, log: log}, nil
}

func (s *Service) Status(ctx context.Context, userID uuid.UUID) (*Connection, error) {
	return s.repo.GetConnection(ctx, userID)
}

// Connect validates the credentials against the server, stores them encrypted,
// and selects a default calendar.
func (s *Service) Connect(ctx context.Context, userID uuid.UUID, login, password string) (*Connection, error) {
	cals, err := discoverCalendars(ctx, login, password)
	if err != nil {
		return nil, err
	}
	if len(cals) == 0 {
		return nil, errors.New("на аккаунте не найдено ни одного календаря")
	}
	def := pickDefault(cals)
	enc, err := s.crypt.encrypt(password)
	if err != nil {
		return nil, err
	}
	return s.repo.Upsert(ctx, userID, "yandex", login, enc, def.URL, def.Name)
}

func (s *Service) ListCalendars(ctx context.Context, userID uuid.UUID) ([]RemoteCalendar, error) {
	conn, err := s.repo.GetConnection(ctx, userID)
	if err != nil {
		return nil, err
	}
	pw, err := s.crypt.decrypt(conn.PasswordEnc)
	if err != nil {
		return nil, err
	}
	return discoverCalendars(ctx, conn.Login, pw)
}

func (s *Service) SelectCalendar(ctx context.Context, userID uuid.UUID, url, name string) error {
	return s.repo.SetCalendar(ctx, userID, url, name)
}

func (s *Service) Disconnect(ctx context.Context, userID uuid.UUID) error {
	return s.repo.Delete(ctx, userID)
}

// Sync runs a full two-way sync for one user and records the outcome.
func (s *Service) Sync(ctx context.Context, userID uuid.UUID) (*SyncResult, error) {
	conn, err := s.repo.GetConnection(ctx, userID)
	if err != nil {
		return nil, err
	}
	if conn.CalendarURL == "" {
		return nil, errors.New("календарь не выбран")
	}
	pw, err := s.crypt.decrypt(conn.PasswordEnc)
	if err != nil {
		return nil, err
	}
	res, syncErr := s.runSync(ctx, conn, pw)
	_ = s.repo.SetSyncMeta(ctx, userID, time.Now(), errString(syncErr))
	return res, syncErr
}

// SyncAll syncs every enabled connection (used by the background worker).
func (s *Service) SyncAll(ctx context.Context) {
	conns, err := s.repo.ListEnabled(ctx)
	if err != nil {
		s.log.Warn("calendar: list connections failed", slog.String("error", err.Error()))
		return
	}
	for i := range conns {
		conn := conns[i]
		pw, err := s.crypt.decrypt(conn.PasswordEnc)
		if err != nil {
			s.log.Warn("calendar: decrypt failed", slog.String("user", conn.UserID.String()))
			continue
		}
		_, syncErr := s.runSync(ctx, &conn, pw)
		_ = s.repo.SetSyncMeta(ctx, conn.UserID, time.Now(), errString(syncErr))
		if syncErr != nil {
			s.log.Warn("calendar: sync failed", slog.String("user", conn.UserID.String()), slog.String("error", syncErr.Error()))
		}
	}
}

func (s *Service) runSync(ctx context.Context, conn *Connection, pw string) (*SyncResult, error) {
	res := &SyncResult{}
	now := time.Now()
	from, to := now.Add(-windowBack), now.Add(windowForward)

	// 1) Propagate local deletions to the server.
	if tombs, err := s.repo.ListTombstones(ctx, conn.UserID); err == nil {
		for _, t := range tombs {
			if err := removeEvent(ctx, conn.Login, pw, t.Href); err != nil {
				if errors.Is(err, ErrInvalidCreds) {
					return res, err
				}
				s.log.Warn("calendar: remove failed", slog.String("error", err.Error()))
				continue // keep the tombstone; retry next sync
			}
			_ = s.repo.DeleteTombstone(ctx, t.ID)
		}
	}

	// 2) Push new / edited local calendar items.
	items, err := s.repo.ItemsToPush(ctx, conn.UserID)
	if err != nil {
		return res, err
	}
	for _, it := range items {
		uid, href, etag, err := pushEvent(ctx, conn.Login, pw, conn.CalendarURL, it)
		if err != nil {
			if errors.Is(err, ErrInvalidCreds) {
				return res, err
			}
			s.log.Warn("calendar: push failed", slog.String("error", err.Error()))
			continue
		}
		if err := s.repo.MarkItemSynced(ctx, conn.UserID, it.ID, uid, href, etag); err != nil {
			return res, err
		}
		res.Pushed++
	}

	// 3) Pull remote events. A pull failure aborts before reconcile — otherwise
	// an empty result would look like "everything was deleted remotely".
	events, err := pullEvents(ctx, conn.Login, pw, conn.CalendarURL, from, to)
	if err != nil {
		return res, err
	}
	keep := make([]string, 0, len(events))
	for _, ev := range events {
		if err := s.repo.UpsertPulledEvent(ctx, conn.UserID, ev); err != nil {
			return res, err
		}
		keep = append(keep, ev.UID)
		res.Pulled++
	}

	// 4) Remove local mirrors of events deleted remotely (within the window).
	if del, err := s.repo.ReconcileDeletions(ctx, conn.UserID, keep, from, to); err == nil {
		res.Deleted = del
	} else {
		s.log.Warn("calendar: reconcile failed", slog.String("error", err.Error()))
	}
	return res, nil
}

func pickDefault(cals []RemoteCalendar) RemoteCalendar {
	for _, c := range cals {
		n := strings.ToLower(c.Name)
		if strings.Contains(n, "событ") || strings.Contains(n, "event") || strings.Contains(n, "default") {
			return c
		}
	}
	return cals[0]
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
