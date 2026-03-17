package httpserver

import (
	"encoding/base64"
	"net/http"
	"strings"
)

// TusCreateGuard validates tus upload create requests (POST /uploads/)
// and enforces that the authenticated user can attach to the specified report.
func TusCreateGuard(deps Deps) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only validate create requests; PATCH/HEAD are resumptions.
			if r.Method != http.MethodPost {
				next.ServeHTTP(w, r)
				return
			}
			p, ok := PrincipalFromContext(r.Context())
			if !ok {
				writeError(w, http.StatusUnauthorized, "unauthorized", "missing or invalid token")
				return
			}
			if deps.ReportService == nil {
				writeError(w, http.StatusInternalServerError, "misconfigured", "service misconfigured")
				return
			}

			meta := parseTusMetadata(r.Header.Get("Upload-Metadata"))
			reportID := strings.TrimSpace(meta["report_id"])
			if reportID == "" {
				writeError(w, http.StatusBadRequest, "validation_error", "report_id is required in Upload-Metadata")
				return
			}

			// Access check: user can only upload to own report; moderator can upload to any.
			if p.Role != "moderator" {
				if _, err := deps.ReportService.GetForUser(r.Context(), p.UserID, reportID); err != nil {
					writeError(w, http.StatusForbidden, "forbidden", "forbidden")
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

func parseTusMetadata(h string) map[string]string {
	out := map[string]string{}
	for _, part := range strings.Split(h, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		k, v, ok := strings.Cut(part, " ")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		if k == "" {
			continue
		}
		if v == "" {
			out[k] = ""
			continue
		}
		decoded, err := base64.StdEncoding.DecodeString(v)
		if err != nil {
			continue
		}
		out[k] = string(decoded)
	}
	return out
}
