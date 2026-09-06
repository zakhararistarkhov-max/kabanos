// Package outbox implements the transactional outbox pattern. Producers write
// a message in the SAME transaction as their business change; the worker later
// claims and delivers it. This gives durable, at-least-once delivery of side
// effects (emails, Telegram messages) without a message broker, and keeps the
// database as the single source of truth even across replica failures.
package outbox

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/kabanos/backend/internal/postgres"
)

// Topics are the delivery channels understood by the worker's dispatcher.
const (
	TopicEmailVerification = "email.verification"
	TopicPasswordReset     = "email.password_reset"
	TopicTelegramDigest    = "telegram.daily_digest"
)

// Message is a claimed outbox row handed to the worker.
type Message struct {
	ID       uuid.UUID
	Topic    string
	Payload  json.RawMessage
	Attempts int
}

type Repo struct{ db *postgres.DB }

func NewRepo(db *postgres.DB) *Repo { return &Repo{db: db} }

// EnqueueTx writes a message inside the caller's transaction. If the
// transaction rolls back, the side effect is never scheduled — atomicity by
// construction.
func EnqueueTx(ctx context.Context, tx pgx.Tx, topic string, payload any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	const q = `INSERT INTO outbox_messages (topic, payload) VALUES ($1, $2)`
	_, err = tx.Exec(ctx, q, topic, raw)
	return err
}

// Claim atomically leases up to `limit` due messages, marking them
// 'processing'. SELECT ... FOR UPDATE SKIP LOCKED lets many worker replicas
// pull disjoint batches concurrently without blocking each other.
func (r *Repo) Claim(ctx context.Context, limit int) ([]Message, error) {
	const q = `
		WITH due AS (
			SELECT id FROM outbox_messages
			WHERE status IN ('pending','failed') AND available_at <= now()
			ORDER BY available_at
			LIMIT $1
			FOR UPDATE SKIP LOCKED
		)
		UPDATE outbox_messages m
		SET status = 'processing', attempts = m.attempts + 1
		FROM due
		WHERE m.id = due.id
		RETURNING m.id, m.topic, m.payload, m.attempts`
	rows, err := r.db.Pool.Query(ctx, q, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []Message
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.Topic, &m.Payload, &m.Attempts); err != nil {
			return nil, err
		}
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}

// Complete marks a message delivered.
func (r *Repo) Complete(ctx context.Context, id uuid.UUID) error {
	const q = `UPDATE outbox_messages SET status = 'done', processed_at = now(), last_error = '' WHERE id = $1`
	_, err := r.db.Pool.Exec(ctx, q, id)
	return err
}

// Fail reschedules a message with exponential backoff, or marks it permanently
// failed after maxAttempts so a poison message cannot loop forever.
func (r *Repo) Fail(ctx context.Context, id uuid.UUID, attempts, maxAttempts int, cause string) error {
	if attempts >= maxAttempts {
		const dead = `UPDATE outbox_messages SET status = 'failed', last_error = $2, available_at = now() + interval '1 hour' WHERE id = $1`
		_, err := r.db.Pool.Exec(ctx, dead, id, cause)
		return err
	}
	backoff := time.Duration(1<<attempts) * 10 * time.Second
	const q = `UPDATE outbox_messages SET status = 'pending', last_error = $2, available_at = now() + $3 WHERE id = $1`
	_, err := r.db.Pool.Exec(ctx, q, id, cause, backoff)
	return err
}
