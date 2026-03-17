package middleware

import (
	"net/http"
	"time"
)

func RateLimit(rps float64, burst int) func(http.Handler) http.Handler {
	// Базовый token bucket per-process (без внешних зависимостей).
	// Для production можно заменить на per-IP или распределенный лимитер (Redis).
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

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			select {
			case <-tokens:
				next.ServeHTTP(w, r)
			default:
				http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
			}
		})
	}
}
