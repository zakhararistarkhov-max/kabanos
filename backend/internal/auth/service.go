package auth

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/kabanos/backend/internal/mailer"
	"github.com/kabanos/backend/internal/outbox"
	"github.com/kabanos/backend/internal/postgres"
	"github.com/kabanos/backend/internal/user"
)

const (
	verificationTTL = 24 * time.Hour
	resetTTL        = 1 * time.Hour
)

// Config carries the auth-relevant settings the service needs.
type Config struct {
	AccessTTL    time.Duration
	RefreshTTL   time.Duration
	JWTSecret    []byte
	Issuer       string
	PublicAppURL string
}

// Service coordinates identity operations. It owns transactions so that a
// business change and its outbox side effect commit atomically.
type Service struct {
	db      *postgres.DB
	users   *user.Repo
	refresh refreshRepo
	verify  singleUseRepo
	reset   singleUseRepo
	tokens  tokenIssuer
	cfg     Config
	log     *slog.Logger
}

func NewService(db *postgres.DB, users *user.Repo, cfg Config, log *slog.Logger) *Service {
	return &Service{
		db:      db,
		users:   users,
		refresh: refreshRepo{db: db},
		verify:  singleUseRepo{db: db, table: "email_verification_tokens"},
		reset:   singleUseRepo{db: db, table: "password_reset_tokens"},
		tokens:  tokenIssuer{secret: cfg.JWTSecret, issuer: cfg.Issuer, accessTTL: cfg.AccessTTL},
		cfg:     cfg,
		log:     log,
	}
}

// Tokens is a freshly minted access/refresh pair.
type Tokens struct {
	AccessToken      string
	AccessExpiresAt  time.Time
	RefreshToken     string
	RefreshExpiresAt time.Time
}

// Session bundles the user with a token pair (returned by register/login).
type Session struct {
	User   *user.User
	Tokens Tokens
}

// Register creates an account, emails a verification link (via the outbox), and
// logs the user in immediately. Registration and the outbox write share one
// transaction, so a verification email is scheduled iff the user was created.
func (s *Service) Register(ctx context.Context, email, password, displayName, userAgent, ip string) (*Session, error) {
	hash, err := HashPassword(password)
	if err != nil {
		return nil, err
	}

	var session *Session
	err = s.inTx(ctx, func(tx pgx.Tx) error {
		u, err := s.users.CreateTx(ctx, tx, email, hash, displayName)
		if err != nil {
			if errors.Is(err, postgres.ErrConflict) {
				return ErrEmailTaken
			}
			return err
		}

		if err := s.scheduleVerification(ctx, tx, u); err != nil {
			return err
		}

		toks, err := s.issueTokens(ctx, tx, u.ID, u.Email, userAgent, ip)
		if err != nil {
			return err
		}
		session = &Session{User: u, Tokens: toks}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return session, nil
}

// Login verifies credentials and issues a token pair. Errors are collapsed to
// ErrInvalidCredentials so attackers cannot distinguish "no such user" from
// "wrong password".
func (s *Service) Login(ctx context.Context, email, password, userAgent, ip string) (*Session, error) {
	u, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			// Still run a hash to keep timing roughly constant.
			_, _ = HashPassword(password)
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}
	ok, err := VerifyPassword(password, u.PasswordHash)
	if err != nil || !ok {
		return nil, ErrInvalidCredentials
	}

	var toks Tokens
	err = s.inTx(ctx, func(tx pgx.Tx) error {
		toks, err = s.issueTokens(ctx, tx, u.ID, u.Email, userAgent, ip)
		return err
	})
	if err != nil {
		return nil, err
	}
	return &Session{User: u, Tokens: toks}, nil
}

// Refresh rotates a refresh token: the presented token is revoked and a new
// pair is issued. Presenting an already-revoked token means the token was
// stolen and replayed → the whole family is revoked (defence in depth).
func (s *Service) Refresh(ctx context.Context, rawRefresh, userAgent, ip string) (Tokens, error) {
	hash := hashSecret(rawRefresh)
	var out Tokens
	err := s.inTx(ctx, func(tx pgx.Tx) error {
		rt, err := s.refresh.getByHash(ctx, tx, hash)
		if err != nil {
			if errors.Is(err, postgres.ErrNotFound) {
				return ErrInvalidToken
			}
			return err
		}
		if rt.RevokedAt != nil {
			// Reuse of a revoked token — revoke everything for this user.
			if err := s.refresh.revokeAllForUser(ctx, tx, rt.UserID); err != nil {
				return err
			}
			s.log.Warn("refresh token reuse detected; revoked token family", slog.String("user_id", rt.UserID.String()))
			return ErrTokenReuse
		}
		if time.Now().After(rt.ExpiresAt) {
			return ErrInvalidToken
		}

		u, err := s.users.GetByID(ctx, rt.UserID)
		if err != nil {
			return err
		}

		toks, newID, err := s.issueTokensReturningID(ctx, tx, u.ID, u.Email, userAgent, ip)
		if err != nil {
			return err
		}
		if err := s.refresh.revoke(ctx, tx, rt.ID, newID); err != nil {
			return err
		}
		out = toks
		return nil
	})
	return out, err
}

// Logout revokes the presented refresh token. Best-effort: an unknown token is
// not an error from the client's perspective.
func (s *Service) Logout(ctx context.Context, rawRefresh string) error {
	return s.refresh.revokeByHash(ctx, s.db.Pool, hashSecret(rawRefresh))
}

// VerifyEmail consumes a verification token and marks the email verified.
func (s *Service) VerifyEmail(ctx context.Context, rawToken string) error {
	hash := hashSecret(rawToken)
	return s.inTx(ctx, func(tx pgx.Tx) error {
		userID, err := s.verify.consume(ctx, tx, hash)
		if err != nil {
			return err // ErrInvalidToken
		}
		return s.users.SetEmailVerified(ctx, tx, userID, time.Now())
	})
}

// RequestPasswordReset emails a reset link if the address exists. It always
// returns nil so the endpoint never reveals whether an email is registered.
func (s *Service) RequestPasswordReset(ctx context.Context, email string) error {
	u, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil // silently succeed
		}
		return err
	}
	return s.inTx(ctx, func(tx pgx.Tx) error {
		if err := s.reset.invalidateAll(ctx, tx, u.ID); err != nil {
			return err
		}
		secret, err := generateSecret()
		if err != nil {
			return err
		}
		if err := s.reset.create(ctx, tx, u.ID, hashSecret(secret), time.Now().Add(resetTTL)); err != nil {
			return err
		}
		link := s.cfg.PublicAppURL + "/reset-password?token=" + secret
		return outbox.EnqueueTx(ctx, tx, outbox.TopicPasswordReset, mailer.PasswordResetPayload{
			To: u.Email, Name: u.DisplayName, Link: link,
		})
	})
}

// ResetPassword consumes a reset token, sets the new password and revokes all
// refresh tokens so any stolen sessions are killed.
func (s *Service) ResetPassword(ctx context.Context, rawToken, newPassword string) error {
	hash, err := HashPassword(newPassword)
	if err != nil {
		return err
	}
	return s.inTx(ctx, func(tx pgx.Tx) error {
		userID, err := s.reset.consume(ctx, tx, hashSecret(rawToken))
		if err != nil {
			return err // ErrInvalidToken
		}
		if err := s.users.UpdatePassword(ctx, tx, userID, hash); err != nil {
			return err
		}
		return s.refresh.revokeAllForUser(ctx, tx, userID)
	})
}

// ResendVerification issues a fresh verification email for an unverified user.
func (s *Service) ResendVerification(ctx context.Context, userID uuid.UUID) error {
	u, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if u.IsEmailVerified() {
		return nil
	}
	return s.inTx(ctx, func(tx pgx.Tx) error {
		if err := s.verify.invalidateAll(ctx, tx, u.ID); err != nil {
			return err
		}
		return s.scheduleVerification(ctx, tx, u)
	})
}

// --- helpers ---

func (s *Service) scheduleVerification(ctx context.Context, tx pgx.Tx, u *user.User) error {
	secret, err := generateSecret()
	if err != nil {
		return err
	}
	if err := s.verify.create(ctx, tx, u.ID, hashSecret(secret), time.Now().Add(verificationTTL)); err != nil {
		return err
	}
	link := s.cfg.PublicAppURL + "/verify-email?token=" + secret
	return outbox.EnqueueTx(ctx, tx, outbox.TopicEmailVerification, mailer.VerificationPayload{
		To: u.Email, Name: u.DisplayName, Link: link,
	})
}

func (s *Service) issueTokens(ctx context.Context, tx pgx.Tx, userID uuid.UUID, email, ua, ip string) (Tokens, error) {
	t, _, err := s.issueTokensReturningID(ctx, tx, userID, email, ua, ip)
	return t, err
}

func (s *Service) issueTokensReturningID(ctx context.Context, tx pgx.Tx, userID uuid.UUID, email, ua, ip string) (Tokens, uuid.UUID, error) {
	access, accessExp, err := s.tokens.IssueAccessToken(userID, email)
	if err != nil {
		return Tokens{}, uuid.Nil, err
	}
	secret, err := generateSecret()
	if err != nil {
		return Tokens{}, uuid.Nil, err
	}
	refreshExp := time.Now().Add(s.cfg.RefreshTTL)
	id, err := s.refresh.create(ctx, tx, userID, hashSecret(secret), refreshExp, ua, ip)
	if err != nil {
		return Tokens{}, uuid.Nil, err
	}
	return Tokens{
		AccessToken:      access,
		AccessExpiresAt:  accessExp,
		RefreshToken:     secret,
		RefreshExpiresAt: refreshExp,
	}, id, nil
}

// inTx runs fn in a transaction, committing on success and rolling back on any
// error or panic.
func (s *Service) inTx(ctx context.Context, fn func(pgx.Tx) error) error {
	tx, err := s.db.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after a successful commit
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
