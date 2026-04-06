package mod_refresh

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"bug-report-service/internal/domain"
	uc "bug-report-service/internal/usecase/mod_refresh"
)

type mockUseCase struct {
	fn func(ctx context.Context, req uc.Request) (uc.Response, error)
}

func (m *mockUseCase) Execute(ctx context.Context, req uc.Request) (uc.Response, error) {
	return m.fn(ctx, req)
}

func TestHandler_Success(t *testing.T) {
	mock := &mockUseCase{fn: func(_ context.Context, _ uc.Request) (uc.Response, error) {
		return uc.Response{
			AccessToken:    "new-access",
			RefreshTokenID: "new-rt-id",
			RefreshToken:   "new-rt-secret",
		}, nil
	}}

	handler := New(mock)

	body := `{"refresh_token_id":"old-id","refresh_token":"old-secret"}`
	req := httptest.NewRequest(http.MethodPost, "/mod/refresh", bytes.NewBufferString(body))
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
	if resp.AccessToken != "new-access" {
		t.Errorf("expected access_token new-access, got %s", resp.AccessToken)
	}
	if resp.RefreshTokenID != "new-rt-id" {
		t.Errorf("expected refresh_token_id new-rt-id, got %s", resp.RefreshTokenID)
	}
}

func TestHandler_InvalidJSON(t *testing.T) {
	mock := &mockUseCase{fn: func(_ context.Context, _ uc.Request) (uc.Response, error) {
		t.Fatal("should not be called")
		return uc.Response{}, nil
	}}

	handler := New(mock)

	req := httptest.NewRequest(http.MethodPost, "/mod/refresh", bytes.NewBufferString(`not-json`))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestHandler_InvalidRefreshToken(t *testing.T) {
	mock := &mockUseCase{fn: func(_ context.Context, _ uc.Request) (uc.Response, error) {
		return uc.Response{}, domain.ErrInvalidRefresh
	}}

	handler := New(mock)

	body := `{"refresh_token_id":"bad-id","refresh_token":"bad-secret"}`
	req := httptest.NewRequest(http.MethodPost, "/mod/refresh", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}
