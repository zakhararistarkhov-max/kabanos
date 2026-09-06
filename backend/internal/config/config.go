// Package config loads and validates all runtime configuration from the
// environment. Configuration is read once at startup and passed explicitly to
// the components that need it (no global state), which keeps the code testable
// and makes the dependency graph obvious.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config is the fully-resolved application configuration.
type Config struct {
	Env             string // "development" | "production" | "test"
	HTTPAddr        string // address the API listens on, e.g. ":8080"
	ShutdownTimeout time.Duration

	Postgres Postgres
	Redis    Redis
	Auth     Auth
	Mailer   Mailer

	// PublicAppURL is the base URL of the frontend, used to build links that
	// are emailed to users (verification / password reset).
	PublicAppURL string

	// CORSAllowedOrigins is the list of origins allowed to call the API from a
	// browser. Empty means "same origin only".
	CORSAllowedOrigins []string

	Telegram Telegram
	S3       S3
}

// S3 configures object storage (MinIO in dev, any S3-compatible store in prod)
// used for dish photos. Endpoint is reachable from the backend; PublicEndpoint
// is what browsers use — presigned URLs are signed for that host so uploads and
// downloads work directly from the client.
type S3 struct {
	Endpoint       string
	PublicEndpoint string
	AccessKey      string
	SecretKey      string
	Bucket         string
	UseSSL         bool
	Region         string
	Enabled        bool
}

// Postgres holds the connection settings for the primary database. Read
// replicas are addressed separately so the application can send read-only
// traffic to them; when unset the primary is used for everything.
type Postgres struct {
	DSN         string
	ReplicaDSNs []string
	MaxConns    int32
	MinConns    int32
	// AutoMigrate applies embedded migrations on startup guarded by an advisory
	// lock so it is safe to run from every replica simultaneously.
	AutoMigrate bool
}

// Redis is used for rate-limiting, ephemeral caching and (optionally) as a
// coordination point between API and worker replicas.
type Redis struct {
	Addr     string
	Password string
	DB       int
}

// Auth configures password hashing and the JWT/refresh-token lifecycle.
type Auth struct {
	// JWTSecret signs short-lived access tokens (HS256). In production this
	// must be a high-entropy secret shared by all API replicas.
	JWTSecret       []byte
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
	Issuer          string
}

// Mailer configures outbound transactional email (SMTP). In development this
// points at Mailhog; in production at a real relay.
type Mailer struct {
	Host      string
	Port      int
	Username  string
	Password  string
	FromName  string
	FromEmail string
	// TLS transport mode: "none" (plaintext, e.g. Mailpit), "starttls"
	// (upgrade on 587) or "tls" (implicit TLS on 465). When empty it is
	// inferred from the port.
	TLS string
}

// TLSMode returns the effective transport mode, inferring from the port when
// not explicitly configured.
func (m Mailer) TLSMode() string {
	switch m.TLS {
	case "none", "starttls", "tls":
		return m.TLS
	}
	switch m.Port {
	case 465:
		return "tls"
	case 587:
		return "starttls"
	default:
		return "none"
	}
}

// Telegram configures the optional bot used for morning digests.
type Telegram struct {
	BotToken string
}

// Load reads configuration from the environment, applies sensible development
// defaults, and validates required fields. It returns an error rather than
// panicking so the caller controls process exit.
func Load() (*Config, error) {
	c := &Config{
		Env:             env("APP_ENV", "development"),
		HTTPAddr:        env("HTTP_ADDR", ":8080"),
		ShutdownTimeout: envDuration("SHUTDOWN_TIMEOUT", 15*time.Second),
		PublicAppURL:    env("PUBLIC_APP_URL", "http://localhost:3000"),
		Postgres: Postgres{
			DSN:         env("POSTGRES_DSN", "postgres://kabanos:kabanos@localhost:5432/kabanos?sslmode=disable"),
			ReplicaDSNs: envList("POSTGRES_REPLICA_DSNS"),
			MaxConns:    int32(envInt("POSTGRES_MAX_CONNS", 10)),
			MinConns:    int32(envInt("POSTGRES_MIN_CONNS", 2)),
			AutoMigrate: envBool("POSTGRES_AUTO_MIGRATE", true),
		},
		Redis: Redis{
			Addr:     env("REDIS_ADDR", "localhost:6379"),
			Password: env("REDIS_PASSWORD", ""),
			DB:       envInt("REDIS_DB", 0),
		},
		Auth: Auth{
			JWTSecret:       []byte(env("JWT_SECRET", "dev-insecure-secret-change-me")),
			AccessTokenTTL:  envDuration("ACCESS_TOKEN_TTL", 15*time.Minute),
			RefreshTokenTTL: envDuration("REFRESH_TOKEN_TTL", 720*time.Hour), // 30d
			Issuer:          env("JWT_ISSUER", "kabanos"),
		},
		Mailer: Mailer{
			Host:      env("SMTP_HOST", "localhost"),
			Port:      envInt("SMTP_PORT", 1025),
			Username:  env("SMTP_USERNAME", ""),
			Password:  env("SMTP_PASSWORD", ""),
			FromName:  env("MAIL_FROM_NAME", "Kabanos"),
			FromEmail: env("MAIL_FROM_EMAIL", "no-reply@kabanos.local"),
			TLS:       env("SMTP_TLS", ""), // "", "none", "starttls" or "tls"
		},
		Telegram: Telegram{
			BotToken: env("TELEGRAM_BOT_TOKEN", ""),
		},
		S3: S3{
			Endpoint:       env("S3_ENDPOINT", "localhost:9000"),
			PublicEndpoint: env("S3_PUBLIC_ENDPOINT", "localhost:9000"),
			AccessKey:      env("S3_ACCESS_KEY", "kabanos"),
			SecretKey:      env("S3_SECRET_KEY", "kabanos-secret"),
			Bucket:         env("S3_BUCKET", "kabanos"),
			UseSSL:         envBool("S3_USE_SSL", false),
			Region:         env("S3_REGION", "us-east-1"),
			Enabled:        envBool("S3_ENABLED", true),
		},
		CORSAllowedOrigins: envList("CORS_ALLOWED_ORIGINS"),
	}
	if len(c.CORSAllowedOrigins) == 0 {
		c.CORSAllowedOrigins = []string{c.PublicAppURL}
	}

	if err := c.validate(); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Config) IsProduction() bool { return c.Env == "production" }

func (c *Config) validate() error {
	if c.Postgres.DSN == "" {
		return fmt.Errorf("POSTGRES_DSN is required")
	}
	if c.IsProduction() {
		if string(c.Auth.JWTSecret) == "dev-insecure-secret-change-me" || len(c.Auth.JWTSecret) < 32 {
			return fmt.Errorf("JWT_SECRET must be set to a strong value (>=32 bytes) in production")
		}
	}
	if c.Auth.AccessTokenTTL <= 0 || c.Auth.RefreshTokenTTL <= c.Auth.AccessTokenTTL {
		return fmt.Errorf("refresh token TTL must be greater than access token TTL")
	}
	return nil
}

// --- small typed env helpers (no external dependency) ---

func env(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) int {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func envBool(key string, def bool) bool {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return def
}

func envDuration(key string, def time.Duration) time.Duration {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}

func envList(key string) []string {
	v, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(v) == "" {
		return nil
	}
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if s := strings.TrimSpace(p); s != "" {
			out = append(out, s)
		}
	}
	return out
}
