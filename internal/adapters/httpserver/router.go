package httpserver

import "net/http"

// NewRouter keeps backward compatibility with server bootstrap.
// At this stage we expose only health/readiness via this function.
func NewRouter(ready Readiness) http.Handler {
	return NewAPI(Deps{Ready: ready})
}
