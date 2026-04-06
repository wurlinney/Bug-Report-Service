package delete_upload

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

type mockUseCase struct {
	fn func(ctx context.Context, uploadSessionID string, storageKey string) (bool, error)
}

func (m *mockUseCase) Execute(ctx context.Context, uploadSessionID string, storageKey string) (bool, error) {
	return m.fn(ctx, uploadSessionID, storageKey)
}

type mockSessionChecker struct {
	fn func(ctx context.Context, id string) (bool, error)
}

func (m *mockSessionChecker) Exists(ctx context.Context, id string) (bool, error) {
	return m.fn(ctx, id)
}

func setup(r *http.Request, sessionID, uploadID string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", sessionID)
	rctx.URLParams.Add("uploadId", uploadID)
	ctx := context.WithValue(r.Context(), chi.RouteCtxKey, rctx)
	return r.WithContext(ctx)
}

func TestHandler_Success(t *testing.T) {
	ucMock := &mockUseCase{fn: func(_ context.Context, _ string, _ string) (bool, error) {
		return true, nil
	}}
	sessMock := &mockSessionChecker{fn: func(_ context.Context, _ string) (bool, error) {
		return true, nil
	}}

	handler := New(ucMock, sessMock)

	req := httptest.NewRequest(http.MethodDelete, "/upload-sessions/sess-1/uploads/upl-1", nil)
	req = setup(req, "sess-1", "upl-1")
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

func TestHandler_SessionNotFound(t *testing.T) {
	ucMock := &mockUseCase{fn: func(_ context.Context, _ string, _ string) (bool, error) {
		t.Fatal("should not be called")
		return false, nil
	}}
	sessMock := &mockSessionChecker{fn: func(_ context.Context, _ string) (bool, error) {
		return false, nil
	}}

	handler := New(ucMock, sessMock)

	req := httptest.NewRequest(http.MethodDelete, "/upload-sessions/bad/uploads/upl-1", nil)
	req = setup(req, "bad", "upl-1")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}
