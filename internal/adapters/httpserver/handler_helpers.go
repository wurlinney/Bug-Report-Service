package httpserver

import "net/http"

func requirePrincipal(w http.ResponseWriter, r *http.Request) (Principal, bool) {
	p, ok := PrincipalFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "missing or invalid token")
		return Principal{}, false
	}
	return p, true
}
