package httpserver

import "net/http"

func ModeratorOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p, ok := PrincipalFromContext(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, "unauthorized", "missing or invalid token")
			return
		}
		if p.Role != "moderator" {
			writeError(w, http.StatusForbidden, "forbidden", "forbidden")
			return
		}
		next.ServeHTTP(w, r)
	})
}
