package api

import (
	"errors"
	"net/http"
	"strings"

	"bug-report-service/internal/api/shared"
)

var ErrUnauthorized = errors.New("unauthorized")

// Principal is re-exported from shared for convenience.
type Principal = shared.Principal

type TokenVerifier interface {
	VerifyAccessToken(token string) (Principal, error)
}

func AuthMiddleware(verifier TokenVerifier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authz := r.Header.Get("Authorization")
			token := strings.TrimPrefix(authz, "Bearer ")
			if token == "" || token == authz || verifier == nil {
				shared.WriteError(w, http.StatusUnauthorized, "unauthorized", "missing or invalid token")
				return
			}
			p, err := verifier.VerifyAccessToken(token)
			if err != nil || p.UserID == "" || p.Role == "" {
				shared.WriteError(w, http.StatusUnauthorized, "unauthorized", "missing or invalid token")
				return
			}
			ctx := shared.PrincipalToContext(r.Context(), p)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func ModeratorOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p, ok := shared.PrincipalFromContext(r.Context())
		if !ok {
			shared.WriteError(w, http.StatusUnauthorized, "unauthorized", "missing or invalid token")
			return
		}
		if p.Role != "moderator" {
			shared.WriteError(w, http.StatusForbidden, "forbidden", "forbidden")
			return
		}
		next.ServeHTTP(w, r)
	})
}
