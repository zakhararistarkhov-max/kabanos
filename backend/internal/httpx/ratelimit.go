package httpx

import (
	"context"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// RateLimiter is a fixed-window limiter backed by Redis so the limit is shared
// across all API replicas (a per-process limiter would let N replicas serve
// N times the intended traffic). It fails open: if Redis is unavailable the
// request is allowed, because availability of the app matters more than a
// perfectly enforced limit.
type RateLimiter struct {
	rdb    *redis.Client
	limit  int
	window time.Duration
}

func NewRateLimiter(rdb *redis.Client, limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{rdb: rdb, limit: limit, window: window}
}

// Middleware limits by a key derived from keyFn (e.g. client IP or user id).
func (rl *RateLimiter) Middleware(prefix string, keyFn func(*http.Request) string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := "rl:" + prefix + ":" + keyFn(r)
			allowed, remaining, reset, err := rl.allow(r.Context(), key)
			if err != nil {
				// Fail open — never let the limiter take down the endpoint.
				next.ServeHTTP(w, r)
				return
			}
			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(rl.limit))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
			w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(reset, 10))
			if !allowed {
				w.Header().Set("Retry-After", strconv.Itoa(int(rl.window.Seconds())))
				Error(w, r, ErrTooManyRequests("too many requests, slow down"))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func (rl *RateLimiter) allow(ctx context.Context, key string) (allowed bool, remaining int, resetUnix int64, err error) {
	ctx, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
	defer cancel()

	pipe := rl.rdb.TxPipeline()
	incr := pipe.Incr(ctx, key)
	// Only set the TTL on the first hit of the window.
	pipe.ExpireNX(ctx, key, rl.window)
	ttl := pipe.TTL(ctx, key)
	if _, err = pipe.Exec(ctx); err != nil {
		return false, 0, 0, err
	}

	count := incr.Val()
	reset := time.Now().Add(ttl.Val()).Unix()
	remaining = rl.limit - int(count)
	if remaining < 0 {
		remaining = 0
	}
	return count <= int64(rl.limit), remaining, reset, nil
}

// ClientIP extracts the best-effort client IP, honouring X-Forwarded-For set by
// the trusted reverse proxy / load balancer in front of the API.
func ClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if i := indexComma(xff); i >= 0 {
			return trimSpace(xff[:i])
		}
		return trimSpace(xff)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func indexComma(s string) int {
	for i := 0; i < len(s); i++ {
		if s[i] == ',' {
			return i
		}
	}
	return -1
}

func trimSpace(s string) string {
	start, end := 0, len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}
