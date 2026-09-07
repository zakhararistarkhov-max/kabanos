package push

import (
	"context"

	"github.com/google/uuid"

	"github.com/kabanos/backend/internal/postgres"
)

type Repo struct{ db *postgres.DB }

func NewRepo(db *postgres.DB) *Repo { return &Repo{db: db} }

// Save records (or refreshes) a subscription, keyed by its unique endpoint.
func (r *Repo) Save(ctx context.Context, userID uuid.UUID, endpoint, p256dh, auth string) error {
	const q = `
		INSERT INTO push_subscriptions (user_id, endpoint, p256dh, auth)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (endpoint) DO UPDATE SET user_id = EXCLUDED.user_id, p256dh = EXCLUDED.p256dh, auth = EXCLUDED.auth`
	_, err := r.db.Pool.Exec(ctx, q, userID, endpoint, p256dh, auth)
	return err
}

func (r *Repo) DeleteByEndpoint(ctx context.Context, endpoint string) error {
	_, err := r.db.Pool.Exec(ctx, `DELETE FROM push_subscriptions WHERE endpoint = $1`, endpoint)
	return err
}

func (r *Repo) ListByUser(ctx context.Context, userID uuid.UUID) ([]Subscription, error) {
	const q = `SELECT id, user_id, endpoint, p256dh, auth, created_at FROM push_subscriptions WHERE user_id = $1`
	rows, err := r.db.Read().Query(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	subs := []Subscription{}
	for rows.Next() {
		var s Subscription
		if err := rows.Scan(&s.ID, &s.UserID, &s.Endpoint, &s.P256dh, &s.Auth, &s.CreatedAt); err != nil {
			return nil, err
		}
		subs = append(subs, s)
	}
	return subs, rows.Err()
}
