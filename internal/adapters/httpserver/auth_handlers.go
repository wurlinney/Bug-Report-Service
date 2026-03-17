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
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "misconfigured"})
			return
		}
		var req authReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "bad_request"})
			return
		}
		req.Email = strings.TrimSpace(req.Email)
		if req.Email == "" || len(req.Password) < 8 {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "validation_error"})
			return
		}

		resp, err := deps.AuthService.Register(r.Context(), auth.RegisterRequest{
			Email:    req.Email,
			Password: req.Password,
		})
		if err != nil {
			switch {
			case err == auth.ErrEmailAlreadyExists:
				writeJSON(w, http.StatusConflict, map[string]any{"error": "email_exists"})
			default:
				writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "auth_error"})
			}
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"access_token":     resp.AccessToken,
			"refresh_token_id": resp.RefreshTokenID,
			"refresh_token":    resp.RefreshToken,
		})
	}
}

func loginHandler(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.AuthService == nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "misconfigured"})
			return
		}
		var req authReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "bad_request"})
			return
		}
		req.Email = strings.TrimSpace(req.Email)
		if req.Email == "" || req.Password == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "validation_error"})
			return
		}

		resp, err := deps.AuthService.Login(r.Context(), auth.LoginRequest{
			Email:    req.Email,
			Password: req.Password,
		})
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "invalid_credentials"})
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
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "misconfigured"})
			return
		}
		var req refreshReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "bad_request"})
			return
		}
		if req.RefreshTokenID == "" || req.RefreshToken == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "validation_error"})
			return
		}

		resp, err := deps.AuthService.Refresh(r.Context(), auth.RefreshRequest{
			RefreshTokenID: req.RefreshTokenID,
			RefreshToken:   req.RefreshToken,
		})
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "invalid_refresh"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"access_token":     resp.AccessToken,
			"refresh_token_id": resp.RefreshTokenID,
			"refresh_token":    resp.RefreshToken,
		})
	}
}
