package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"

	"bug-report-service/internal/application/auth"
)

type authReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type refreshReq struct {
	RefreshTokenID string `json:"refresh_token_id"`
	RefreshToken   string `json:"refresh_token"`
}

func registerHandler(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.AuthService == nil {
			writeError(w, http.StatusInternalServerError, "misconfigured", "service misconfigured")
			return
		}
		var req authReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "bad_request", "invalid json body")
			return
		}
		req.Email = strings.TrimSpace(req.Email)
		if req.Email == "" || len(req.Password) < 8 {
			writeError(w, http.StatusBadRequest, "validation_error", "email required, password min 8 chars")
			return
		}

		resp, err := deps.AuthService.Register(r.Context(), auth.RegisterRequest{
			Email:    req.Email,
			Password: req.Password,
		})
		if err != nil {
			switch {
			case err == auth.ErrEmailAlreadyExists:
				writeError(w, http.StatusConflict, "email_exists", "email already exists")
			default:
				writeError(w, http.StatusInternalServerError, "internal_error", "internal error")
			}
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{
			"access_token":     resp.AccessToken,
			"refresh_token_id": resp.RefreshTokenID,
			"refresh_token":    resp.RefreshToken,
		})
	}
}

func loginHandler(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.AuthService == nil {
			writeError(w, http.StatusInternalServerError, "misconfigured", "service misconfigured")
			return
		}
		var req authReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "bad_request", "invalid json body")
			return
		}
		req.Email = strings.TrimSpace(req.Email)
		if req.Email == "" || req.Password == "" {
			writeError(w, http.StatusBadRequest, "validation_error", "email and password are required")
			return
		}

		resp, err := deps.AuthService.Login(r.Context(), auth.LoginRequest{
			Email:    req.Email,
			Password: req.Password,
		})
		if err != nil {
			writeError(w, http.StatusUnauthorized, "invalid_credentials", "invalid credentials")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"access_token":     resp.AccessToken,
			"refresh_token_id": resp.RefreshTokenID,
			"refresh_token":    resp.RefreshToken,
		})
	}
}

func refreshHandler(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.AuthService == nil {
			writeError(w, http.StatusInternalServerError, "misconfigured", "service misconfigured")
			return
		}
		var req refreshReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "bad_request", "invalid json body")
			return
		}
		if req.RefreshTokenID == "" || req.RefreshToken == "" {
			writeError(w, http.StatusBadRequest, "validation_error", "refresh_token_id and refresh_token are required")
			return
		}

		resp, err := deps.AuthService.Refresh(r.Context(), auth.RefreshRequest{
			RefreshTokenID: req.RefreshTokenID,
			RefreshToken:   req.RefreshToken,
		})
		if err != nil {
			writeError(w, http.StatusUnauthorized, "invalid_refresh", "invalid refresh token")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"access_token":     resp.AccessToken,
			"refresh_token_id": resp.RefreshTokenID,
			"refresh_token":    resp.RefreshToken,
		})
	}
}
