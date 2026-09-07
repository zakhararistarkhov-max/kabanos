// Command worker delivers transactional-outbox messages (emails, Telegram
// digests). It is a separate process so it can be scaled and deployed
// independently of the API. Multiple worker replicas can run at once: the
// outbox Claim uses FOR UPDATE SKIP LOCKED, so they cooperate without
// double-delivery within a batch.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	// Embed tzdata for the static runtime image (see cmd/api/main.go).
	_ "time/tzdata"

	"github.com/kabanos/backend/internal/config"
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

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			processBatch(ctx, box, d, logger)
		case <-remTicker.C:
			processReminders(ctx, remRepo, pushSvc, logger)
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
