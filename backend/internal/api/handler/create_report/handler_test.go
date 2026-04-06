package create_report

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"bug-report-service/internal/domain"
	"bug-report-service/internal/domain/report"
	uc "bug-report-service/internal/usecase/create_report"
)

type mockUseCase struct {
	fn func(ctx context.Context, req uc.Request) (report.Report, error)
}

func (m *mockUseCase) Execute(ctx context.Context, req uc.Request) (report.Report, error) {
	return m.fn(ctx, req)
}

func TestHandler_Success(t *testing.T) {
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	mock := &mockUseCase{fn: func(_ context.Context, req uc.Request) (report.Report, error) {
		return report.Report{
			ID:           "r-1",
			ReporterName: req.ReporterName,
			Description:  req.Description,
			Status:       "new",
			CreatedAt:    now,
			UpdatedAt:    now,
		}, nil
	}}

	handler := New(mock)

	body := `{"reporter_name":"Alice","description":"Something broke"}`
	req := httptest.NewRequest(http.MethodPost, "/reports", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rr.Code)
	}

	var resp responseBody
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.ID != "r-1" {
		t.Errorf("expected id r-1, got %s", resp.ID)
	}
	if resp.ReporterName != "Alice" {
		t.Errorf("expected reporter_name Alice, got %s", resp.ReporterName)
	}
	if resp.Status != "new" {
		t.Errorf("expected status new, got %s", resp.Status)
	}
}

func TestHandler_InvalidJSON(t *testing.T) {
	mock := &mockUseCase{fn: func(_ context.Context, _ uc.Request) (report.Report, error) {
		t.Fatal("should not be called")
		return report.Report{}, nil
	}}

	handler := New(mock)

	req := httptest.NewRequest(http.MethodPost, "/reports", bytes.NewBufferString(`{invalid`))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestHandler_MissingReporterName(t *testing.T) {
	mock := &mockUseCase{fn: func(_ context.Context, _ uc.Request) (report.Report, error) {
		return report.Report{}, domain.ErrBadInput
	}}

	handler := New(mock)

	body := `{"reporter_name":"","description":"Something broke"}`
	req := httptest.NewRequest(http.MethodPost, "/reports", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}
