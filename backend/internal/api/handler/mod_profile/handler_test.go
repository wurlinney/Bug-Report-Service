package mod_profile

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"bug-report-service/internal/api/shared"
	"bug-report-service/internal/domain"
	uc "bug-report-service/internal/usecase/mod_profile"
)

type mockUseCase struct {
	fn func(ctx context.Context, moderatorID string) (uc.Profile, error)
}

func (m *mockUseCase) Execute(ctx context.Context, moderatorID string) (uc.Profile, error) {
	return m.fn(ctx, moderatorID)
}

func withPrincipal(r *http.Request, p shared.Principal) *http.Request {
	ctx := shared.PrincipalToContext(r.Context(), p)
	return r.WithContext(ctx)
}

func TestHandler_Success(t *testing.T) {
	mock := &mockUseCase{fn: func(_ context.Context, id string) (uc.Profile, error) {
		return uc.Profile{
			ID:        id,
			Name:      "Admin",
			Email:     "admin@test.com",
			CreatedAt: 1700000000,
			UpdatedAt: 1700000000,
		}, nil
	}}

	handler := New(mock)

	req := httptest.NewRequest(http.MethodGet, "/mod/profile", nil)
	req = withPrincipal(req, shared.Principal{UserID: "mod-1", Role: "moderator"})
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp responseBody
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.ID != "mod-1" {
		t.Errorf("expected id mod-1, got %s", resp.ID)
	}
	if resp.Name != "Admin" {
		t.Errorf("expected name Admin, got %s", resp.Name)
	}
}

func TestHandler_NotFound(t *testing.T) {
	mock := &mockUseCase{fn: func(_ context.Context, _ string) (uc.Profile, error) {
		return uc.Profile{}, domain.ErrNotFound
	}}

	handler := New(mock)

	req := httptest.NewRequest(http.MethodGet, "/mod/profile", nil)
	req = withPrincipal(req, shared.Principal{UserID: "unknown", Role: "moderator"})
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}

func TestHandler_NoPrincipal(t *testing.T) {
	mock := &mockUseCase{fn: func(_ context.Context, _ string) (uc.Profile, error) {
		t.Fatal("should not be called")
		return uc.Profile{}, nil
	}}

	handler := New(mock)

	req := httptest.NewRequest(http.MethodGet, "/mod/profile", nil)
	// No principal
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}
