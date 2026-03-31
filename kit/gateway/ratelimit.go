package gateway

import (
	"net/http"
	"sync"

	"golang.org/x/time/rate"
)

// RateLimitConfig configures the rate limiter.
type RateLimitConfig struct {
	// RequestsPerSecond is the sustained rate limit.
	RequestsPerSecond float64
	// Burst is the maximum burst size (token bucket capacity).
	Burst int
}

// RateLimiter returns a middleware that limits requests using a token bucket.
// Returns 429 Too Many Requests when the limit is exceeded.
func RateLimiter(cfg RateLimitConfig) Middleware {
	limiter := rate.NewLimiter(rate.Limit(cfg.RequestsPerSecond), cfg.Burst)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !limiter.Allow() {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte(`{"error":"rate limit exceeded"}`))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// PerIPRateLimiter returns a middleware that limits requests per client IP.
func PerIPRateLimiter(cfg RateLimitConfig) Middleware {
	var mu sync.Mutex
	limiters := make(map[string]*rate.Limiter)

	getLimiter := func(ip string) *rate.Limiter {
		mu.Lock()
		defer mu.Unlock()
		if l, ok := limiters[ip]; ok {
			return l
		}
		l := rate.NewLimiter(rate.Limit(cfg.RequestsPerSecond), cfg.Burst)
		limiters[ip] = l
		return l
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := r.RemoteAddr
			if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
				ip = fwd
			}
			if !getLimiter(ip).Allow() {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte(`{"error":"rate limit exceeded"}`))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
