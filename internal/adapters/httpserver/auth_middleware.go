package httpserver

import (
	"context"
	"net/http"
	"strings"
)

type principalKey struct{}

func AuthMiddleware(verifier TokenVerifier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authz := r.Header.Get("Authorization")
			token := strings.TrimPrefix(authz, "Bearer ")
			if token == "" || token == authz || verifier == nil {
				writeError(w, http.StatusUnauthorized, "unauthorized", "missing or invalid token")
				return
			}
			p, err := verifier.VerifyAccessToken(token)
			if err != nil || p.UserID == "" || p.Role == "" {
				writeError(w, http.StatusUnauthorized, "unauthorized", "missing or invalid token")
				return
			}
			ctx := context.WithValue(r.Context(), principalKey{}, p)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func PrincipalFromContext(ctx context.Context) (Principal, bool) {
	p, ok := ctx.Value(principalKey{}).(Principal)
	return p, ok
}
