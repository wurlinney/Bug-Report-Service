package get_report

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"bug-report-service/internal/api/shared"
	"bug-report-service/internal/domain"
	"bug-report-service/internal/domain/report"

	"github.com/go-chi/chi/v5"
)

type mockUseCase struct {
	fn func(ctx context.Context, actorRole string, reportID string) (report.Report, error)
}

func (m *mockUseCase) Execute(ctx context.Context, actorRole string, reportID string) (report.Report, error) {
	return m.fn(ctx, actorRole, reportID)
}

func withPrincipal(r *http.Request, p shared.Principal) *http.Request {
	ctx := shared.PrincipalToContext(r.Context(), p)
	return r.WithContext(ctx)
}

func withChiParam(r *http.Request, key, value string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, value)
	ctx := context.WithValue(r.Context(), chi.RouteCtxKey, rctx)
	return r.WithContext(ctx)
}

func TestHandler_Success(t *testing.T) {
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	mock := &mockUseCase{fn: func(_ context.Context, _ string, id string) (report.Report, error) {
		return report.Report{
			ID:           id,
			ReporterName: "Alice",
			Description:  "Bug",
			Status:       "new",
			CreatedAt:    now,
			UpdatedAt:    now,
		}, nil
	}}

	handler := New(mock)

	req := httptest.NewRequest(http.MethodGet, "/reports/r-1", nil)
	req = withChiParam(req, "id", "r-1")
	req = withPrincipal(req, shared.Principal{UserID: "1", Role: "moderator"})
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp responseBody
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.ID != "r-1" {
		t.Errorf("expected id r-1, got %s", resp.ID)
	}
}

func TestHandler_NotFound(t *testing.T) {
	mock := &mockUseCase{fn: func(_ context.Context, _ string, _ string) (report.Report, error) {
		return report.Report{}, domain.ErrNotFound
	}}

	handler := New(mock)

	req := httptest.NewRequest(http.MethodGet, "/reports/unknown", nil)
	req = withChiParam(req, "id", "unknown")
	req = withPrincipal(req, shared.Principal{UserID: "1", Role: "moderator"})
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}

func TestHandler_NoPrincipal(t *testing.T) {
	mock := &mockUseCase{fn: func(_ context.Context, _ string, _ string) (report.Report, error) {
		t.Fatal("should not be called")
		return report.Report{}, nil
	}}

	handler := New(mock)

	req := httptest.NewRequest(http.MethodGet, "/reports/r-1", nil)
	req = withChiParam(req, "id", "r-1")
	// No principal set
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}
