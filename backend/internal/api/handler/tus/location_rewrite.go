package tus

import (
	"net/http"
	"strings"
)

// withTusLocationRewrite wraps a handler and rewrites the Location header
// returned by tusd so that it uses the public-facing base URL instead of the
// internal one. The basePath is the public prefix (e.g. "/api/upload").
func WithTusLocationRewrite(basePath string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rw := &locationRewriteRW{
			ResponseWriter: w,
			basePath:       basePath,
			request:        r,
		}
		next.ServeHTTP(rw, r)
	})
}

// locationRewriteRW is a ResponseWriter wrapper that rewrites the Location
// header so the tus client sees the correct public URL.
type locationRewriteRW struct {
	http.ResponseWriter
	basePath string
	request  *http.Request
}

func (rw *locationRewriteRW) WriteHeader(code int) {
	loc := rw.Header().Get("Location")
	if loc != "" {
		rw.Header().Set("Location", rw.rewrite(loc))
	}
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *locationRewriteRW) rewrite(location string) string {
	// Extract just the file ID from the location returned by tusd.
	// tusd typically returns something like "http://host/files/abc123".
	// We want to rewrite it to "{basePath}/abc123".
	parts := strings.Split(location, "/")
	if len(parts) == 0 {
		return location
	}
	fileID := parts[len(parts)-1]
	if fileID == "" {
		return location
	}

	scheme := "http"
	if rw.request.TLS != nil {
		scheme = "https"
	}
	if fwd := rw.request.Header.Get("X-Forwarded-Proto"); fwd != "" {
		scheme = fwd
	}

	host := rw.request.Host
	basePath := strings.TrimRight(rw.basePath, "/")

	return scheme + "://" + host + basePath + "/" + fileID
}
