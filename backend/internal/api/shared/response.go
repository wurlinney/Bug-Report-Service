package shared

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"bug-report-service/internal/domain"
)

type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message,omitempty"`
}

func WriteJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func WriteError(w http.ResponseWriter, status int, code string, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(ErrorResponse{
		Error: ErrorBody{
			Code:    code,
			Message: message,
		},
	})
}

func WriteDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrBadInput):
		WriteError(w, http.StatusBadRequest, "validation_error", "invalid parameters")
	case errors.Is(err, domain.ErrNotFound):
		WriteError(w, http.StatusNotFound, "not_found", "not found")
	case errors.Is(err, domain.ErrForbidden):
		WriteError(w, http.StatusForbidden, "forbidden", "forbidden")
	case errors.Is(err, domain.ErrInvalidCredentials):
		WriteError(w, http.StatusUnauthorized, "invalid_credentials", "invalid credentials")
	case errors.Is(err, domain.ErrInvalidRefresh):
		WriteError(w, http.StatusUnauthorized, "invalid_refresh", "invalid refresh token")
	default:
		WriteError(w, http.StatusInternalServerError, "internal_error", "internal error")
	}
}

func ParseInt(s string, def int) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}

func ParseUnixSeconds(s string) time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil || n <= 0 {
		return time.Time{}
	}
	return time.Unix(n, 0).UTC()
}

func TimePtr(v time.Time) *time.Time {
	if v.IsZero() {
		return nil
	}
	return &v
}
