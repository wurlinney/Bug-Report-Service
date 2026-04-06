package tus

import (
	"net/http"

	"bug-report-service/internal/api/shared"
)

// TusCreateGuard is a middleware that validates POST requests to the tus
// endpoint. It parses the Upload-Metadata header and checks that the
// upload_session_id references an existing session.
func TusCreateGuard(sessions SessionChecker) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				next.ServeHTTP(w, r)
				return
			}

			meta := parseTusMetadata(r.Header.Get("Upload-Metadata"))
			sessionID := meta["upload_session_id"]
			if sessionID == "" {
				shared.WriteError(w, http.StatusBadRequest, "validation_error", "missing upload_session_id in metadata")
				return
			}

			exists, err := sessions.Exists(r.Context(), sessionID)
			if err != nil {
				shared.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to check session")
				return
			}
			if !exists {
				shared.WriteError(w, http.StatusBadRequest, "validation_error", "upload session not found")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
