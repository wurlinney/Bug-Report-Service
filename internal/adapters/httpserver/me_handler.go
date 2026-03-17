package httpserver

import (
	"net/http"

	"bug-report-service/internal/application/user"
)

func meHandler(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p, ok := PrincipalFromContext(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, "unauthorized", "missing or invalid token")
			return
		}
		if deps.UserService == nil {
			writeError(w, http.StatusInternalServerError, "misconfigured", "service misconfigured")
			return
		}

		profile, err := deps.UserService.GetProfile(r.Context(), p.UserID)
		if err != nil {
			if err == user.ErrUserNotFound {
				writeError(w, http.StatusUnauthorized, "unauthorized", "missing or invalid token")
				return
			}
			writeError(w, http.StatusInternalServerError, "internal_error", "internal error")
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"id":         profile.ID,
			"email":      profile.Email,
			"role":       profile.Role,
			"created_at": profile.CreatedAt,
			"updated_at": profile.UpdatedAt,
		})
	}
}
