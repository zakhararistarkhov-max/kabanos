package push

import (
	"context"
	"encoding/json"
	"log/slog"

	webpush "github.com/SherClockHolmes/webpush-go"
	"github.com/google/uuid"

	"github.com/kabanos/backend/internal/config"
)

type Service struct {
	repo *Repo
	cfg  config.VAPID
	log  *slog.Logger
}

func NewService(repo *Repo, cfg config.VAPID, log *slog.Logger) *Service {
	return &Service{repo: repo, cfg: cfg, log: log}
}

func (s *Service) Enabled() bool     { return s.cfg.Enabled() }
func (s *Service) PublicKey() string { return s.cfg.PublicKey }

func (s *Service) Subscribe(ctx context.Context, userID uuid.UUID, endpoint, p256dh, auth string) error {
	return s.repo.Save(ctx, userID, endpoint, p256dh, auth)
}

func (s *Service) Unsubscribe(ctx context.Context, endpoint string) error {
	return s.repo.DeleteByEndpoint(ctx, endpoint)
}

// Send delivers a notification to every device the user has subscribed. Dead
// subscriptions (404/410 from the push service) are pruned. Returns how many
// devices were reached.
func (s *Service) Send(ctx context.Context, userID uuid.UUID, n Notification) (int, error) {
	if !s.Enabled() {
		return 0, nil
	}
	subs, err := s.repo.ListByUser(ctx, userID)
	if err != nil {
		return 0, err
	}
	payload, err := json.Marshal(n)
	if err != nil {
		return 0, err
	}

	sent := 0
	for _, sub := range subs {
		resp, err := webpush.SendNotificationWithContext(ctx, payload, &webpush.Subscription{
			Endpoint: sub.Endpoint,
			Keys:     webpush.Keys{P256dh: sub.P256dh, Auth: sub.Auth},
		}, &webpush.Options{
			Subscriber:      s.cfg.WebPushSubscriber(),
			VAPIDPublicKey:  s.cfg.PublicKey,
			VAPIDPrivateKey: s.cfg.PrivateKey,
			TTL:             60,
		})
		if err != nil {
			s.log.Warn("push send failed", slog.String("error", err.Error()))
			continue
		}
		status := resp.StatusCode
		resp.Body.Close()
		switch {
		case status == 404 || status == 410:
			// Subscription is gone — drop it so we stop trying.
			_ = s.repo.DeleteByEndpoint(ctx, sub.Endpoint)
		case status < 300:
			sent++
		default:
			s.log.Warn("push service rejected", slog.Int("status", status))
		}
	}
	return sent, nil
}
