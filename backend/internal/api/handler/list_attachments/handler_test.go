package list_attachments

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"bug-report-service/internal/api/shared"
	"bug-report-service/internal/domain/attachment"
	uc "bug-report-service/internal/usecase/list_attachments"

	"github.com/go-chi/chi/v5"
)

type mockUseCase struct {
	fn func(ctx context.Context, req uc.Request) ([]uc.AttachmentWithURL, error)
}

func (m *mockUseCase) Execute(ctx context.Context, req uc.Request) ([]uc.AttachmentWithURL, error) {
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
	mock := &mockUseCase{fn: func(_ context.Context, _ uc.Request) ([]uc.AttachmentWithURL, error) {
		return []uc.AttachmentWithURL{
			{
				Attachment: attachment.Attachment{
					ID:              1,
					ReportID:        "r-1",
					UploadSessionID: "sess-1",
					FileName:        "bug.png",
					ContentType:     "image/png",
					FileSize:        1024,
					StorageKey:      "tus/abc",
					CreatedAt:       now,
				},
				SignedURL: "https://example.com/signed",
			},
		}, nil
	}}

	handler := New(mock)

	req := httptest.NewRequest(http.MethodGet, "/reports/r-1/attachments", nil)
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
	if len(resp.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(resp.Items))
	}
	if resp.Items[0].FileName != "bug.png" {
		t.Errorf("expected file_name bug.png, got %s", resp.Items[0].FileName)
	}
	if resp.Items[0].DownloadURL != "https://example.com/signed" {
		t.Errorf("expected download_url, got %s", resp.Items[0].DownloadURL)
	}
}

func TestHandler_NoPrincipal(t *testing.T) {
	mock := &mockUseCase{fn: func(_ context.Context, _ uc.Request) ([]uc.AttachmentWithURL, error) {
		t.Fatal("should not be called")
		return nil, nil
	}}

	handler := New(mock)

	req := httptest.NewRequest(http.MethodGet, "/reports/r-1/attachments", nil)
	// Set chi param but no principal
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "r-1")
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}
