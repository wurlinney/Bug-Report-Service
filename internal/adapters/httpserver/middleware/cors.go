package middleware

import (
	"net/http"
	"strings"
)

func CORS(allowedOrigins []string) func(http.Handler) http.Handler {
	allowAll := len(allowedOrigins) == 1 && allowedOrigins[0] == "*"
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, o := range allowedOrigins {
		allowed[strings.TrimSpace(o)] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" {
				if allowAll {
					w.Header().Set("Access-Control-Allow-Origin", "*")
				} else {
					if _, ok := allowed[origin]; ok {
						w.Header().Set("Access-Control-Allow-Origin", origin)
						w.Header().Set("Vary", "Origin")
					}
				}

				// NOTE: tus protocol relies on custom headers (Tus-Resumable, Upload-*).
				// If frontend and backend are on different origins, preflight must allow them.
				w.Header().Set("Access-Control-Allow-Headers", strings.Join([]string{
					"Authorization",
					"Content-Type",
					"X-Request-Id",
					"Idempotency-Key",
					"Tus-Resumable",
					"Upload-Length",
					"Upload-Offset",
					"Upload-Metadata",
					"Upload-Defer-Length",
					"Upload-Concat",
				}, ", "))
				w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PATCH,PUT,DELETE,OPTIONS,HEAD")
				w.Header().Set("Access-Control-Expose-Headers", strings.Join([]string{
					"Location",
					"Tus-Resumable",
					"Upload-Offset",
					"Upload-Length",
				}, ", "))
			}

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
