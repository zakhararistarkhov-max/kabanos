// Command worker delivers transactional-outbox messages (emails, Telegram
// digests). It is a separate process so it can be scaled and deployed
// independently of the API. Multiple worker replicas can run at once: the
// outbox Claim uses FOR UPDATE SKIP LOCKED, so they cooperate without
// double-delivery within a batch.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	// Embed tzdata for the static runtime image (see cmd/api/main.go).
	_ "time/tzdata"

	"github.com/google/uuid"

	"github.com/kabanos/backend/internal/calendar"
	"github.com/kabanos/backend/internal/config"
	"github.com/kabanos/backend/internal/fasting"
	"github.com/kabanos/backend/internal/mailer"
	"github.com/kabanos/backend/internal/notify"
	"github.com/kabanos/backend/internal/observability"
	"github.com/kabanos/backend/internal/outbox"
	"github.com/kabanos/backend/internal/postgres"
	"github.com/kabanos/backend/internal/push"
	"github.com/kabanos/backend/internal/reminders"
)

const (
	pollInterval     = 2 * time.Second
	reminderInterval = time.Minute
	calendarInterval = 10 * time.Minute
	batchSize        = 20
	maxAttempts      = 5
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		observability.NewLogger(os.Getenv("APP_ENV"), "kabanos-worker").Error("failed to load config", slog.String("error", err.Error()))
		os.Exit(1)
	}
	logger := observability.NewLogger(cfg.Env, "kabanos-worker")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, cfg, logger); err != nil {
		logger.Error("worker exited with error", slog.String("error", err.Error()))
		os.Exit(1)
	}
	logger.Info("worker stopped cleanly")
}

func run(ctx context.Context, cfg *config.Config, logger *slog.Logger) error {
	// Migrations are idempotent and advisory-locked, so it is safe for the
	// worker to ensure the schema too — it can then boot before the API.
	if cfg.Postgres.AutoMigrate {
		if err := postgres.Migrate(ctx, cfg.Postgres.DSN, logger); err != nil {
			return err
		}
	}
	db, err := postgres.New(ctx, cfg.Postgres, logger)
	if err != nil {
		return err
	}
	defer db.Close()

	box := outbox.NewRepo(db)
	mail := mailer.New(cfg.Mailer)
	tg := notify.NewTelegram(cfg.Telegram.BotToken)
	d := &dispatcher{mail: mail, tg: tg}

	pushSvc := push.NewService(push.NewRepo(db), cfg.VAPID, logger)
	remRepo := reminders.NewRepo(db)
	fastingRepo := fasting.NewRepo(db)
	calSvc, err := calendar.NewService(calendar.NewRepo(db), calendar.DeriveKey(cfg.CalendarEncKey, cfg.Auth.JWTSecret), logger)
	if err != nil {
		return fmt.Errorf("calendar service: %w", err)
	}

	// Make the effective mail destination unmistakable in the logs — the #1
	// source of "why didn't my email arrive?" is SMTP still pointing at the dev
	// catcher (Mailpit), which never delivers to real inboxes.
	logger.Info("outbound email transport",
		slog.String("smtp", fmt.Sprintf("%s:%d", cfg.Mailer.Host, cfg.Mailer.Port)),
		slog.String("tls", cfg.Mailer.TLSMode()),
		slog.String("from", cfg.Mailer.FromEmail),
	)
	if isDevMailCatcher(cfg.Mailer) {
		logger.Warn("SMTP is a DEV mail catcher — email is captured locally and NOT delivered to real inboxes; set SMTP_HOST/PORT/USERNAME/PASSWORD in .env for real delivery")
	}

	logger.Info("worker started",
		slog.Duration("poll_interval", pollInterval),
		slog.Bool("push_enabled", pushSvc.Enabled()),
	)
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	remTicker := time.NewTicker(reminderInterval)
	defer remTicker.Stop()
	calTicker := time.NewTicker(calendarInterval)
	defer calTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			processBatch(ctx, box, d, logger)
		case <-remTicker.C:
			processReminders(ctx, remRepo, pushSvc, logger)
			processFastingSchedules(ctx, fastingRepo, pushSvc, logger)
		case <-calTicker.C:
			calSvc.SyncAll(ctx)
		}
	}
}

// processReminders evaluates every enabled reminder and pushes the due ones.
// ClaimFire makes delivery safe across multiple worker replicas (only one wins
// each occurrence).
func processReminders(ctx context.Context, repo *reminders.Repo, pushSvc *push.Service, logger *slog.Logger) {
	if !pushSvc.Enabled() {
		return
	}
	list, err := repo.ListEnabled(ctx)
	if err != nil {
		logger.Error("list reminders", slog.String("error", err.Error()))
		return
	}
	now := time.Now()
	for _, rem := range list {
		if !rem.DueAt(now) {
			continue
		}
		if rem.Condition != "" {
			ok, err := conditionMet(ctx, repo, rem, now)
			if err != nil {
				logger.Error("reminder condition", slog.String("error", err.Error()))
				continue
			}
			if !ok {
				continue // goal already met — skip without claiming, retry later
			}
		}
		claimed, err := repo.ClaimFire(ctx, rem.ID, rem.LastFiredAt, now)
		if err != nil {
			logger.Error("claim reminder", slog.String("error", err.Error()))
			continue
		}
		if !claimed {
			continue // another replica handled it
		}
		if _, err := pushSvc.Send(ctx, rem.UserID, push.Notification{Title: rem.Title, Body: rem.Body, URL: rem.URL}); err != nil {
			logger.Error("send reminder push", slog.String("error", err.Error()))
		}
	}
}

// conditionMet checks a reminder's suppression condition against live data.
func conditionMet(ctx context.Context, repo *reminders.Repo, rem reminders.Reminder, now time.Time) (bool, error) {
	loc, err := time.LoadLocation(rem.Timezone)
	if err != nil {
		loc = time.UTC
	}
	local := now.In(loc)
	switch rem.Condition {
	case "water_below_goal":
		from := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)
		return repo.WaterBelowGoal(ctx, rem.UserID, from, from.AddDate(0, 0, 1))
	case "meds_due":
		return repo.MedsDue(ctx, rem.UserID, local.Format("2006-01-02"))
	}
	return true, nil
}

// fastingFireTolerance bounds how late an auto-start may fire (e.g. if the
// worker was briefly down) before the day's slot is considered missed.
const fastingFireTolerance = 30 * time.Minute

// processFastingSchedules auto-starts scheduled fasts and pushes the start and
// hourly-countdown notifications. It sends at most one notification per user per
// tick; the ClaimNotify/ClaimAutoStart guards make it safe across replicas.
func processFastingSchedules(ctx context.Context, repo *fasting.Repo, pushSvc *push.Service, logger *slog.Logger) {
	if !pushSvc.Enabled() {
		return
	}
	rows, err := repo.ListEnabledSchedules(ctx)
	if err != nil {
		logger.Error("list fasting schedules", slog.String("error", err.Error()))
		return
	}
	now := time.Now()
	for _, row := range rows {
		loc, err := time.LoadLocation(row.Schedule.Timezone)
		if err != nil {
			loc = time.UTC
		}
		local := now.In(loc)

		active, err := repo.GetActive(ctx, row.UserID)
		hasActive := err == nil
		if err != nil && !errors.Is(err, postgres.ErrNotFound) {
			logger.Error("fasting active lookup", slog.String("error", err.Error()))
			continue
		}

		// 1) Auto-start the fast at the scheduled local time (once per day).
		if !hasActive && row.Schedule.AutoStart {
			slot := time.Date(local.Year(), local.Month(), local.Day(), row.Schedule.StartHour, row.Schedule.StartMinute, 0, 0, loc)
			today := local.Format("2006-01-02")
			startedToday := row.AutoStartedOn != nil && row.AutoStartedOn.Format("2006-01-02") == today
			if !startedToday && !now.Before(slot) && now.Sub(slot) <= fastingFireTolerance {
				won, cerr := repo.ClaimAutoStart(ctx, row.UserID, today)
				switch {
				case cerr != nil:
					logger.Error("claim fasting auto-start", slog.String("error", cerr.Error()))
				case won:
					sess, serr := repo.Create(ctx, row.UserID, slot, row.FastingHours)
					if serr != nil {
						// A concurrent manual start likely won the unique index — skip.
						logger.Warn("fasting auto-start create", slog.String("error", serr.Error()))
					} else {
						active, hasActive = sess, true
					}
				}
			}
		}
		if !hasActive {
			continue
		}

		// 2) Notify: the start push, then one hourly countdown per elapsed hour.
		start := active.StartedAt
		goalH := int(active.GoalHours)
		if goalH < 1 {
			goalH = 1
		}

		// New fast (anchor changed) → reset the cursor and send the start push.
		if row.NotifyAnchor == nil || !row.NotifyAnchor.Equal(start) {
			won, cerr := repo.ClaimNotify(ctx, row.UserID, row.NotifyAnchor, row.NotifyLastHour, start, 0)
			if cerr != nil {
				logger.Error("claim fasting notify (start)", slog.String("error", cerr.Error()))
			} else if won && row.Schedule.NotifyStart {
				end := start.Add(time.Duration(active.GoalHours * float64(time.Hour)))
				sendFastingPush(ctx, pushSvc, logger, row.UserID, push.Notification{
					Title: "Голодание началось",
					Body:  fmt.Sprintf("Цель — %s. Продержитесь до %s.", fmtHours(active.GoalHours), end.In(loc).Format("15:04")),
					URL:   "/fasting",
				})
			}
			continue
		}

		// Hourly countdown at each whole elapsed hour, up to the goal.
		elapsed := int(now.Sub(start).Hours())
		if row.NotifyLastHour < goalH && elapsed >= 1 && elapsed > row.NotifyLastHour {
			target := elapsed
			if target > goalH {
				target = goalH
			}
			won, cerr := repo.ClaimNotify(ctx, row.UserID, row.NotifyAnchor, row.NotifyLastHour, start, target)
			if cerr != nil {
				logger.Error("claim fasting notify (hourly)", slog.String("error", cerr.Error()))
			} else if won && row.Schedule.NotifyHourly {
				var n push.Notification
				if remaining := goalH - target; remaining <= 0 {
					n = push.Notification{Title: "Цель достигнута! 🎉", Body: fmt.Sprintf("%s голодания позади. Можно открывать окно еды.", fmtHours(active.GoalHours)), URL: "/fasting"}
				} else {
					n = push.Notification{Title: fmt.Sprintf("Голодание: осталось %d ч", remaining), Body: fmt.Sprintf("Прошло %d ч из %d. Держитесь!", target, goalH), URL: "/fasting"}
				}
				sendFastingPush(ctx, pushSvc, logger, row.UserID, n)
			}
		}
	}
}

func sendFastingPush(ctx context.Context, pushSvc *push.Service, logger *slog.Logger, userID uuid.UUID, n push.Notification) {
	if _, err := pushSvc.Send(ctx, userID, n); err != nil {
		logger.Error("send fasting push", slog.String("error", err.Error()))
	}
}

// fmtHours renders a fasting length like "18 ч" or "18,5 ч" (Russian decimal).
func fmtHours(h float64) string {
	if h == math.Trunc(h) {
		return fmt.Sprintf("%d ч", int(h))
	}
	return strings.Replace(fmt.Sprintf("%.1f ч", h), ".", ",", 1)
}

func processBatch(ctx context.Context, box *outbox.Repo, d *dispatcher, logger *slog.Logger) {
	msgs, err := box.Claim(ctx, batchSize)
	if err != nil {
		logger.Error("claim outbox batch", slog.String("error", err.Error()))
		return
	}
	for _, m := range msgs {
		// Bound each delivery so one slow relay cannot stall the batch loop.
		deliverCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
		err := d.dispatch(deliverCtx, m)
		cancel()

		if err != nil {
			logger.Error("deliver outbox message",
				slog.String("id", m.ID.String()),
				slog.String("topic", m.Topic),
				slog.Int("attempt", m.Attempts),
				slog.String("error", err.Error()),
			)
			if ferr := box.Fail(ctx, m.ID, m.Attempts, maxAttempts, err.Error()); ferr != nil {
				logger.Error("mark outbox failed", slog.String("error", ferr.Error()))
			}
			continue
		}
		if cerr := box.Complete(ctx, m.ID); cerr != nil {
			logger.Error("mark outbox done", slog.String("error", cerr.Error()))
		}
	}
}

// dispatcher routes an outbox message to the correct channel by topic.
type dispatcher struct {
	mail *mailer.Mailer
	tg   *notify.Telegram
}

func (d *dispatcher) dispatch(ctx context.Context, m outbox.Message) error {
	switch m.Topic {
	case outbox.TopicEmailVerification:
		var p mailer.VerificationPayload
		if err := json.Unmarshal(m.Payload, &p); err != nil {
			return err
		}
		return d.mail.SendVerification(ctx, p)

	case outbox.TopicPasswordReset:
		var p mailer.PasswordResetPayload
		if err := json.Unmarshal(m.Payload, &p); err != nil {
			return err
		}
		return d.mail.SendPasswordReset(ctx, p)

	case outbox.TopicTelegramDigest:
		var p notify.DigestPayload
		if err := json.Unmarshal(m.Payload, &p); err != nil {
			return err
		}
		return d.tg.Send(ctx, p.ChatID, p.Message)

	default:
		// Unknown topic is a programming error; fail it so it is visible but
		// does not spin forever (Fail will dead-letter after maxAttempts).
		return errUnknownTopic(m.Topic)
	}
}

type errUnknownTopic string

func (e errUnknownTopic) Error() string { return "unknown outbox topic: " + string(e) }

// isDevMailCatcher reports whether the SMTP target is a local dev catcher
// (Mailpit/Mailhog) rather than a relay that reaches real inboxes.
func isDevMailCatcher(m config.Mailer) bool {
	switch m.Host {
	case "mailpit", "mailhog", "localhost", "127.0.0.1", "":
		return true
	}
	return m.Port == 1025
}
