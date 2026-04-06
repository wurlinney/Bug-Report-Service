package create_upload_session

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"bug-report-service/internal/domain/uploadsession"
)

type mockUseCase struct {
	fn func(ctx context.Context) (uploadsession.UploadSession, error)
}

func (m *mockUseCase) Execute(ctx context.Context) (uploadsession.UploadSession, error) {
	return m.fn(ctx)
}

func TestHandler_Success(t *testing.T) {
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	mock := &mockUseCase{fn: func(_ context.Context) (uploadsession.UploadSession, error) {
		return uploadsession.UploadSession{
			ID:        "sess-1",
			CreatedAt: now,
		}, nil
	}}

	handler := New(mock)

	req := httptest.NewRequest(http.MethodPost, "/upload-sessions", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rr.Code)
	}

	var resp responseBody
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.ID != "sess-1" {
		t.Errorf("expected id sess-1, got %s", resp.ID)
	}
	if resp.CreatedAt != now.Unix() {
		t.Errorf("expected created_at %d, got %d", now.Unix(), resp.CreatedAt)
	}
}
