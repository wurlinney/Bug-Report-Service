package create_note

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"bug-report-service/internal/api/shared"
	"bug-report-service/internal/domain/note"
	uc "bug-report-service/internal/usecase/create_note"

	"github.com/go-chi/chi/v5"
)

type mockUseCase struct {
	fn func(ctx context.Context, req uc.Request) (note.Note, error)
}

func (m *mockUseCase) Execute(ctx context.Context, req uc.Request) (note.Note, error) {
	return m.fn(ctx, req)
}

func setup(r *http.Request, reportID string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", reportID)
	ctx := context.WithValue(r.Context(), chi.RouteCtxKey, rctx)
	ctx = shared.PrincipalToContext(ctx, shared.Principal{UserID: "mod-1", Role: "moderator"})
	return r.WithContext(ctx)
}

func TestHandler_Success(t *testing.T) {
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	mock := &mockUseCase{fn: func(_ context.Context, req uc.Request) (note.Note, error) {
		return note.Note{
			ID:                "n-1",
			ReportID:          req.ReportID,
			AuthorModeratorID: req.ActorID,
			Text:              req.Text,
			CreatedAt:         now,
		}, nil
	}}

	handler := New(mock)

	body := `{"text":"This is a note"}`
	req := httptest.NewRequest(http.MethodPost, "/reports/r-1/notes", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = setup(req, "r-1")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rr.Code)
	}

	var resp responseBody
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.ID != "n-1" {
		t.Errorf("expected id n-1, got %s", resp.ID)
	}
	if resp.ReportID != "r-1" {
		t.Errorf("expected report_id r-1, got %s", resp.ReportID)
	}
	if resp.Text != "This is a note" {
		t.Errorf("expected text 'This is a note', got %s", resp.Text)
	}
}

func TestHandler_InvalidJSON(t *testing.T) {
	mock := &mockUseCase{fn: func(_ context.Context, _ uc.Request) (note.Note, error) {
		t.Fatal("should not be called")
		return note.Note{}, nil
	}}

	handler := New(mock)

	req := httptest.NewRequest(http.MethodPost, "/reports/r-1/notes", bytes.NewBufferString(`{bad`))
	req = setup(req, "r-1")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}
