package httpserver

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"bug-report-service/internal/application/auth"
	"bug-report-service/internal/application/moderator"
)

type modLoginReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type modRefreshReq struct {
	RefreshTokenID string `json:"refresh_token_id"`
	RefreshToken   string `json:"refresh_token"`
}

func modLoginHandler(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.ModAuthService == nil {
			writeError(w, http.StatusInternalServerError, "misconfigured", "service misconfigured")
			return
		}
		var req modLoginReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "bad_request", "invalid json body")
			return
		}
		req.Email = strings.TrimSpace(req.Email)
		if req.Email == "" || req.Password == "" {
			writeError(w, http.StatusBadRequest, "validation_error", "email and password are required")
			return
		}

		resp, err := deps.ModAuthService.Login(r.Context(), auth.LoginRequest{
			Email:    req.Email,
			Password: req.Password,
		})
		if err != nil {
			if errors.Is(err, auth.ErrInvalidCredentials) {
				writeError(w, http.StatusUnauthorized, "invalid_credentials", "invalid credentials")
				return
			}
			writeError(w, http.StatusInternalServerError, "internal_error", "internal error")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"access_token":     resp.AccessToken,
			"refresh_token_id": resp.RefreshTokenID,
			"refresh_token":    resp.RefreshToken,
		})
	}
}

func modRefreshHandler(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.ModAuthService == nil {
			writeError(w, http.StatusInternalServerError, "misconfigured", "service misconfigured")
			return
		}
		var req modRefreshReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "bad_request", "invalid json body")
			return
		}
		if strings.TrimSpace(req.RefreshTokenID) == "" || strings.TrimSpace(req.RefreshToken) == "" {
			writeError(w, http.StatusBadRequest, "validation_error", "refresh_token_id and refresh_token are required")
			return
		}

		resp, err := deps.ModAuthService.Refresh(r.Context(), auth.RefreshRequest{
			RefreshTokenID: req.RefreshTokenID,
			RefreshToken:   req.RefreshToken,
		})
		if err != nil {
			if errors.Is(err, auth.ErrInvalidRefreshToken) {
				writeError(w, http.StatusUnauthorized, "invalid_refresh", "invalid refresh token")
				return
			}
			writeError(w, http.StatusInternalServerError, "internal_error", "internal error")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"access_token":     resp.AccessToken,
			"refresh_token_id": resp.RefreshTokenID,
			"refresh_token":    resp.RefreshToken,
		})
	}
}

func modMeHandler(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p, ok := PrincipalFromContext(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, "unauthorized", "missing or invalid token")
			return
		}
		if p.Role != "moderator" {
			writeError(w, http.StatusForbidden, "forbidden", "forbidden")
			return
		}
		if deps.ModeratorService == nil {
			writeError(w, http.StatusInternalServerError, "misconfigured", "service misconfigured")
			return
		}

		profile, err := deps.ModeratorService.GetProfile(r.Context(), p.UserID)
		if err != nil {
			if errors.Is(err, moderator.ErrModeratorNotFound) {
				writeError(w, http.StatusUnauthorized, "unauthorized", "missing or invalid token")
				return
			}
			writeError(w, http.StatusInternalServerError, "internal_error", "internal error")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"id":         profile.ID,
			"name":       profile.Name,
			"email":      profile.Email,
			"created_at": profile.CreatedAt,
			"updated_at": profile.UpdatedAt,
		})
	}
}
