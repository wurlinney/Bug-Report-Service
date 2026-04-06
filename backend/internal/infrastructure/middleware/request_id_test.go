package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestID_GeneratesWhenMissing(t *testing.T) {
	handler := RequestID()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rid := GetRequestID(r.Context())
		if rid == "" {
			t.Fatal("expected request id in context, got empty string")
		}
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	got := rec.Header().Get("X-Request-Id")
	if got == "" {
		t.Fatal("expected X-Request-Id response header to be set")
	}
	if len(got) != 32 {
		t.Fatalf("expected 32-char hex id, got %q (len %d)", got, len(got))
	}
}

func TestRequestID_UsesProvidedHeader(t *testing.T) {
	const want = "my-custom-id-123"

	var ctxRID string
	handler := RequestID()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctxRID = GetRequestID(r.Context())
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-Id", want)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Header().Get("X-Request-Id") != want {
		t.Fatalf("response header: got %q, want %q", rec.Header().Get("X-Request-Id"), want)
	}
	if ctxRID != want {
		t.Fatalf("context value: got %q, want %q", ctxRID, want)
	}
}

func TestGetRequestID_ReturnsValueFromContext(t *testing.T) {
	const want = "ctx-id-456"
	ctx := context.WithValue(context.Background(), RequestIDKey, want)
	got := GetRequestID(ctx)
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestGetRequestID_ReturnsEmptyForBareContext(t *testing.T) {
	got := GetRequestID(context.Background())
	if got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
}
