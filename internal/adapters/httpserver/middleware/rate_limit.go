package middleware

import (
	"encoding/json"
	"net/http"
	"time"
)

type RateLimiter struct {
	tokens chan struct{}
	ticker *time.Ticker
}

func NewRateLimiter(rps float64, burst int) *RateLimiter {
	if rps <= 0 {
		rps = 1
	}
	if burst <= 0 {
		burst = 1
	}

	tokens := make(chan struct{}, burst)
	for i := 0; i < burst; i++ {
		tokens <- struct{}{}
	}

	interval := time.Duration(float64(time.Second) / rps)
	if interval <= 0 {
		interval = time.Nanosecond
	}
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			select {
			case tokens <- struct{}{}:
			default:
			}
		}
	}()

	return &RateLimiter{tokens: tokens, ticker: ticker}
}

func (rl *RateLimiter) Stop() {
	rl.ticker.Stop()
}

func (rl *RateLimiter) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			select {
			case <-rl.tokens:
				next.ServeHTTP(w, r)
			default:
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"error": map[string]any{
						"code":    "rate_limit",
						"message": "too many requests",
					},
				})
			}
		})
	}
}

// RateLimit is a convenience wrapper that creates a RateLimiter and returns
// its middleware. The ticker goroutine cannot be stopped; prefer NewRateLimiter
// for production use.
func RateLimit(rps float64, burst int) func(http.Handler) http.Handler {
	return NewRateLimiter(rps, burst).Middleware()
}
