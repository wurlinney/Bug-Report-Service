package list_notes

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"bug-report-service/internal/api/shared"
	"bug-report-service/internal/domain/note"
	uc "bug-report-service/internal/usecase/list_notes"

	"github.com/go-chi/chi/v5"
)

type mockUseCase struct {
	fn func(ctx context.Context, req uc.Request) ([]note.Note, int, error)
}

func (m *mockUseCase) Execute(ctx context.Context, req uc.Request) ([]note.Note, int, error) {
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
	mock := &mockUseCase{fn: func(_ context.Context, _ uc.Request) ([]note.Note, int, error) {
		return []note.Note{
			{
				ID:                "n-1",
				ReportID:          "r-1",
				AuthorModeratorID: "mod-1",
				Text:              "note text",
				CreatedAt:         now,
			},
		}, 1, nil
	}}

	handler := New(mock)

	req := httptest.NewRequest(http.MethodGet, "/reports/r-1/notes", nil)
	req = setup(req, "r-1")
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
	if resp.Items[0].ID != "n-1" {
		t.Errorf("expected note id n-1, got %s", resp.Items[0].ID)
	}
}
