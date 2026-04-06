package list_reports

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
	uc "bug-report-service/internal/usecase/list_reports"
)

type mockUseCase struct {
	fn func(ctx context.Context, req uc.Request) ([]report.Report, int, error)
}

func (m *mockUseCase) Execute(ctx context.Context, req uc.Request) ([]report.Report, int, error) {
	return m.fn(ctx, req)
}

func withPrincipal(r *http.Request, p shared.Principal) *http.Request {
	ctx := shared.PrincipalToContext(r.Context(), p)
	return r.WithContext(ctx)
}

func TestHandler_Success(t *testing.T) {
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	mock := &mockUseCase{fn: func(_ context.Context, _ uc.Request) ([]report.Report, int, error) {
		return []report.Report{
			{
				ID:           "r-1",
				ReporterName: "Alice",
				Description:  "Bug 1",
				Status:       "new",
				CreatedAt:    now,
				UpdatedAt:    now,
			},
		}, 1, nil
	}}

	handler := New(mock)

	req := httptest.NewRequest(http.MethodGet, "/reports?limit=10&offset=0", nil)
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
	if resp.Total != 1 {
		t.Errorf("expected total 1, got %d", resp.Total)
	}
	if len(resp.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(resp.Items))
	}
	if resp.Items[0].ID != "r-1" {
		t.Errorf("expected item id r-1, got %s", resp.Items[0].ID)
	}
}

func TestHandler_Forbidden(t *testing.T) {
	mock := &mockUseCase{fn: func(_ context.Context, _ uc.Request) ([]report.Report, int, error) {
		return nil, 0, domain.ErrForbidden
	}}

	handler := New(mock)

	req := httptest.NewRequest(http.MethodGet, "/reports", nil)
	req = withPrincipal(req, shared.Principal{UserID: "1", Role: "viewer"})
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rr.Code)
	}
}
