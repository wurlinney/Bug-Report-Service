package api

import (
	"net/http"
	"time"

	"bug-report-service/internal/api/shared"
)

// Re-export shared helpers for use within the api package.

func WriteJSON(w http.ResponseWriter, code int, v any) {
	shared.WriteJSON(w, code, v)
}

func WriteError(w http.ResponseWriter, status int, code string, message string) {
	shared.WriteError(w, status, code, message)
}

func WriteDomainError(w http.ResponseWriter, err error) {
	shared.WriteDomainError(w, err)
}

func ParseInt(s string, def int) int {
	return shared.ParseInt(s, def)
}

func ParseUnixSeconds(s string) time.Time {
	return shared.ParseUnixSeconds(s)
}

func TimePtr(v time.Time) *time.Time {
	return shared.TimePtr(v)
}

func PrincipalFromContext(r *http.Request) (Principal, bool) {
	return shared.PrincipalFromContext(r.Context())
}
