// Package postgres provides the database connection pools and migration
// runner. It exposes a small DB wrapper that separates the read/write primary
// pool from optional read-replica pools so read-heavy endpoints can be scaled
// horizontally without touching the primary.
package postgres

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kabanos/backend/internal/config"
)

// DB owns the primary (read/write) pool and any read-replica pools.
type DB struct {
	Pool     *pgxpool.Pool   // primary: use for writes and read-your-write reads
	replicas []*pgxpool.Pool // optional read-only replicas
	log      *slog.Logger
}

// New builds the connection pools from configuration and verifies connectivity.
func New(ctx context.Context, cfg config.Postgres, log *slog.Logger) (*DB, error) {
	primary, err := newPool(ctx, cfg.DSN, cfg.MaxConns, cfg.MinConns)
	if err != nil {
		return nil, fmt.Errorf("connect primary: %w", err)
	}

	db := &DB{Pool: primary, log: log}
	for _, dsn := range cfg.ReplicaDSNs {
		p, err := newPool(ctx, dsn, cfg.MaxConns, cfg.MinConns)
		if err != nil {
			// A replica being down must never take the service down; degrade to
			// serving reads from the primary and log loudly.
			log.Error("replica connection failed, falling back to primary for its share of reads", slog.String("error", err.Error()))
			continue
		}
		db.replicas = append(db.replicas, p)
	}
	return db, nil
}

func newPool(ctx context.Context, dsn string, maxConns, minConns int32) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}
	poolCfg.MaxConns = maxConns
	poolCfg.MinConns = minConns
	poolCfg.MaxConnLifetime = time.Hour
	poolCfg.MaxConnIdleTime = 30 * time.Minute
	poolCfg.HealthCheckPeriod = time.Minute
	// Fail fast on a dead connection rather than hanging a request.
	poolCfg.ConnConfig.ConnectTimeout = 5 * time.Second

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, err
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}

// Read returns a pool suitable for read-only queries. When replicas are
// configured it load-balances across them; otherwise it returns the primary.
// Callers that require read-your-own-write consistency must use Pool directly.
func (db *DB) Read() *pgxpool.Pool {
	if len(db.replicas) == 0 {
		return db.Pool
	}
	return db.replicas[rand.IntN(len(db.replicas))]
}

// Ping verifies the primary is reachable; used by the readiness probe.
func (db *DB) Ping(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return db.Pool.Ping(ctx)
}

// Close releases all pools.
func (db *DB) Close() {
	db.Pool.Close()
	for _, r := range db.replicas {
		r.Close()
	}
}
