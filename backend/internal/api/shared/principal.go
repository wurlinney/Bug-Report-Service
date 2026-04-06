package shared

import (
	"context"
	"net/http"
)

type Principal struct {
	UserID string
	Role   string
}

type principalKey struct{}

func PrincipalToContext(ctx context.Context, p Principal) context.Context {
	return context.WithValue(ctx, principalKey{}, p)
}

func PrincipalFromContext(ctx context.Context) (Principal, bool) {
	p, ok := ctx.Value(principalKey{}).(Principal)
	return p, ok
}

func RequirePrincipal(w http.ResponseWriter, r *http.Request) (Principal, bool) {
	p, ok := PrincipalFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusUnauthorized, "unauthorized", "missing or invalid token")
		return Principal{}, false
	}
	return p, true
}
