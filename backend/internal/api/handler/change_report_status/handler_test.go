package change_report_status

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"bug-report-service/internal/api/shared"
	"bug-report-service/internal/domain"
	ucMeta "bug-report-service/internal/usecase/change_report_meta"
	ucStatus "bug-report-service/internal/usecase/change_report_status"

	"github.com/go-chi/chi/v5"
)

type mockStatusUC struct {
	fn func(ctx context.Context, req ucStatus.Request) error
}

func (m *mockStatusUC) Execute(ctx context.Context, req ucStatus.Request) error {
	return m.fn(ctx, req)
}

type mockMetaUC struct {
	fn func(ctx context.Context, req ucMeta.Request) error
}

func (m *mockMetaUC) Execute(ctx context.Context, req ucMeta.Request) error {
	return m.fn(ctx, req)
}

func setup(r *http.Request, reportID string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", reportID)
	ctx := context.WithValue(r.Context(), chi.RouteCtxKey, rctx)
	ctx = shared.PrincipalToContext(ctx, shared.Principal{UserID: "1", Role: "moderator"})
	return r.WithContext(ctx)
}

func TestHandler_Success(t *testing.T) {
	statusMock := &mockStatusUC{fn: func(_ context.Context, _ ucStatus.Request) error {
		return nil
	}}
	metaMock := &mockMetaUC{fn: func(_ context.Context, _ ucMeta.Request) error {
		return nil
	}}

	handler := New(statusMock, metaMock)

	body := `{"status":"in_review"}`
	req := httptest.NewRequest(http.MethodPatch, "/reports/r-1/status", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = setup(req, "r-1")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["status"] != "ok" {
		t.Errorf("expected status ok, got %s", resp["status"])
	}
}

func TestHandler_InvalidJSON(t *testing.T) {
	statusMock := &mockStatusUC{fn: func(_ context.Context, _ ucStatus.Request) error {
		t.Fatal("should not be called")
		return nil
	}}

	handler := New(statusMock, nil)

	req := httptest.NewRequest(http.MethodPatch, "/reports/r-1/status", bytes.NewBufferString(`{bad`))
	req = setup(req, "r-1")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestHandler_DomainError(t *testing.T) {
	statusMock := &mockStatusUC{fn: func(_ context.Context, _ ucStatus.Request) error {
		return domain.ErrBadInput
	}}

	handler := New(statusMock, nil)

	body := `{"status":"invalid_status"}`
	req := httptest.NewRequest(http.MethodPatch, "/reports/r-1/status", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = setup(req, "r-1")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}
