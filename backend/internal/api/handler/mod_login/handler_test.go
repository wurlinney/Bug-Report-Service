package mod_login

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"bug-report-service/internal/domain"
	uc "bug-report-service/internal/usecase/mod_login"
)

type mockUseCase struct {
	fn func(ctx context.Context, req uc.Request) (uc.Response, error)
}

func (m *mockUseCase) Execute(ctx context.Context, req uc.Request) (uc.Response, error) {
	return m.fn(ctx, req)
}

func TestHandler_Success(t *testing.T) {
	mock := &mockUseCase{fn: func(_ context.Context, req uc.Request) (uc.Response, error) {
		return uc.Response{
			AccessToken:    "access-token",
			RefreshTokenID: "rt-id",
			RefreshToken:   "rt-secret",
		}, nil
	}}

	handler := New(mock)

	body := `{"email":"admin@test.com","password":"secret"}`
	req := httptest.NewRequest(http.MethodPost, "/mod/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp responseBody
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.AccessToken != "access-token" {
		t.Errorf("expected access_token access-token, got %s", resp.AccessToken)
	}
	if resp.RefreshTokenID != "rt-id" {
		t.Errorf("expected refresh_token_id rt-id, got %s", resp.RefreshTokenID)
	}
}

func TestHandler_InvalidJSON(t *testing.T) {
	mock := &mockUseCase{fn: func(_ context.Context, _ uc.Request) (uc.Response, error) {
		t.Fatal("should not be called")
		return uc.Response{}, nil
	}}

	handler := New(mock)

	req := httptest.NewRequest(http.MethodPost, "/mod/login", bytes.NewBufferString(`{bad`))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestHandler_MissingEmail(t *testing.T) {
	mock := &mockUseCase{fn: func(_ context.Context, _ uc.Request) (uc.Response, error) {
		return uc.Response{}, domain.ErrInvalidCredentials
	}}

	handler := New(mock)

	body := `{"email":"","password":"secret"}`
	req := httptest.NewRequest(http.MethodPost, "/mod/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// The use case returns ErrInvalidCredentials which maps to 401,
	// but from the handler's perspective missing email is a domain error.
	// WriteDomainError maps ErrInvalidCredentials -> 401.
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestHandler_InvalidCredentials(t *testing.T) {
	mock := &mockUseCase{fn: func(_ context.Context, _ uc.Request) (uc.Response, error) {
		return uc.Response{}, domain.ErrInvalidCredentials
	}}

	handler := New(mock)

	body := `{"email":"admin@test.com","password":"wrong"}`
	req := httptest.NewRequest(http.MethodPost, "/mod/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}
