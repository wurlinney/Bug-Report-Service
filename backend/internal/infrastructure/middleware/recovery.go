package middleware

import (
	"encoding/json"
	"net/http"
	"runtime/debug"

	"bug-report-service/internal/infrastructure/logger"
)

func Recovery(log logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					log.Error("panic recovered",
						"request_id", GetRequestID(r.Context()),
						"panic", rec,
						"stack", string(debug.Stack()),
					)
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)
					_ = json.NewEncoder(w).Encode(map[string]any{
						"error": map[string]any{
							"code":    "internal_error",
							"message": "internal server error",
						},
					})
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
