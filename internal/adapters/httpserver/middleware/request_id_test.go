package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestID_GeneratesWhenMissing(t *testing.T) {
	h := RequestID()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if GetRequestID(r.Context()) == "" {
			t.Fatalf("expected request id in context")
		}
		w.WriteHeader(http.StatusOK)
	}))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://example.test/", nil)
	h.ServeHTTP(rr, req)

	if rr.Header().Get("X-Request-Id") == "" {
		t.Fatalf("expected X-Request-Id response header")
	}
}

func TestRequestID_UsesIncomingHeader(t *testing.T) {
	const rid = "abc-123"
	h := RequestID()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := GetRequestID(r.Context()); got != rid {
			t.Fatalf("expected %q, got %q", rid, got)
		}
		w.WriteHeader(http.StatusOK)
	}))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://example.test/", nil)
	req.Header.Set("X-Request-Id", rid)
	h.ServeHTTP(rr, req)

	if got := rr.Header().Get("X-Request-Id"); got != rid {
		t.Fatalf("expected response header %q, got %q", rid, got)
	}
}
