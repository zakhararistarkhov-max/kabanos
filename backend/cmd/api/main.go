// Command api is the Kabanos HTTP API server. It is stateless: all state lives
// in Postgres/Redis, so any number of replicas can run behind a load balancer.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	// Embed the timezone database so time.LoadLocation works in the distroless
	// (static, no OS zoneinfo) runtime image.
	_ "time/tzdata"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/kabanos/backend/internal/auth"
	"github.com/kabanos/backend/internal/config"
	"github.com/kabanos/backend/internal/httpx"
	"github.com/kabanos/backend/internal/nutrition"
	"github.com/kabanos/backend/internal/observability"
	"github.com/kabanos/backend/internal/postgres"
	"github.com/kabanos/backend/internal/redisx"
	"github.com/kabanos/backend/internal/storage"
	"github.com/kabanos/backend/internal/user"
	"github.com/kabanos/backend/internal/water"
	"github.com/kabanos/backend/internal/weight"
)

// weightAdapter adapts the weight repo to nutrition.WeightProvider (used for
// MET-based calorie calculations), keeping the nutrition package decoupled.
type weightAdapter struct{ repo *weight.Repo }

func (a weightAdapter) LatestKg(ctx context.Context, userID uuid.UUID) (float64, bool, error) {
	e, err := a.repo.Latest(ctx, userID)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return 0, false, nil
		}
		return 0, false, err
	}
	return e.WeightKg, true, nil
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		observability.NewLogger(os.Getenv("APP_ENV")).Error("failed to load config", slog.String("error", err.Error()))
		os.Exit(1)
	}
	logger := observability.NewLogger(cfg.Env)

	// Root context cancelled on SIGINT/SIGTERM → triggers graceful shutdown.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, cfg, logger); err != nil {
		logger.Error("server exited with error", slog.String("error", err.Error()))
		os.Exit(1)
	}
	logger.Info("server stopped cleanly")
}

func run(ctx context.Context, cfg *config.Config, logger *slog.Logger) error {
	// --- infrastructure ---
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

	rdb, err := redisx.New(ctx, cfg.Redis)
	if err != nil {
		return err
	}
	defer rdb.Close()

	store, err := storage.New(ctx, cfg.S3)
	if err != nil {
		// Object storage is non-critical for the rest of the app: log and run
		// with image uploads disabled rather than failing startup.
		logger.Error("object storage unavailable; dish photos disabled", slog.String("error", err.Error()))
		store, _ = storage.New(ctx, config.S3{Enabled: false})
	}

	// --- repositories ---
	users := user.NewRepo(db)
	waterRepo := water.NewRepo(db)
	weightRepo := weight.NewRepo(db)
	dishRepo := nutrition.NewDishRepo(db)
	logRepo := nutrition.NewLogRepo(db)

	// --- services ---
	authSvc := auth.NewService(db, users, auth.Config{
		AccessTTL:    cfg.Auth.AccessTokenTTL,
		RefreshTTL:   cfg.Auth.RefreshTokenTTL,
		JWTSecret:    cfg.Auth.JWTSecret,
		Issuer:       cfg.Auth.Issuer,
		PublicAppURL: cfg.PublicAppURL,
	}, logger)
	waterSvc := water.NewService(waterRepo)
	weightSvc := weight.NewService(weightRepo, users)
	nutritionSvc := nutrition.NewService(dishRepo, logRepo, store, weightAdapter{repo: weightRepo})

	// --- handlers ---
	authH := auth.NewHandler(authSvc, users)
	waterH := water.NewHandler(waterSvc)
	weightH := weight.NewHandler(weightSvc)
	nutritionH := nutrition.NewHandler(nutritionSvc)

	// --- rate limiters (shared across replicas via Redis) ---
	authLimiter := httpx.NewRateLimiter(rdb, 20, time.Minute) // brute-force protection
	apiLimiter := httpx.NewRateLimiter(rdb, 300, time.Minute) // general per-user budget

	// --- router ---
	r := chi.NewRouter()
	r.Use(httpx.RequestContext(logger))
	r.Use(httpx.AccessLog)
	r.Use(httpx.Recoverer)
	r.Use(httpx.CORS(cfg.CORSAllowedOrigins))

	// Health probes (unauthenticated, unlimited).
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	r.Get("/readyz", func(w http.ResponseWriter, req *http.Request) {
		if err := db.Ping(req.Context()); err != nil {
			httpx.JSON(w, http.StatusServiceUnavailable, map[string]string{"status": "db_unavailable"})
			return
		}
		if err := rdb.Ping(req.Context()).Err(); err != nil {
			httpx.JSON(w, http.StatusServiceUnavailable, map[string]string{"status": "redis_unavailable"})
			return
		}
		httpx.JSON(w, http.StatusOK, map[string]string{"status": "ready"})
	})

	r.Route("/api/v1", func(r chi.Router) {
		// Public auth endpoints, IP rate-limited.
		r.Group(func(r chi.Router) {
			r.Use(authLimiter.Middleware("auth", httpx.ClientIP))
			r.Mount("/auth", authH.PublicRoutes())
		})

		// Authenticated endpoints, per-user rate-limited.
		r.Group(func(r chi.Router) {
			r.Use(authSvc.Authenticator())
			r.Use(apiLimiter.Middleware("api", func(req *http.Request) string {
				return auth.UserID(req.Context()).String()
			}))
			r.Get("/me", authH.Me)
			r.Patch("/me", authH.UpdateMe)
			r.Post("/auth/resend-verification", authH.ResendVerification)
			r.Mount("/water", waterH.Routes())
			r.Mount("/weight", weightH.Routes())
			r.Mount("/nutrition", nutritionH.Routes())
		})
	})

	srv := httpx.NewServer(cfg.HTTPAddr, r, logger)
	return srv.Run(ctx, cfg.ShutdownTimeout)
}
