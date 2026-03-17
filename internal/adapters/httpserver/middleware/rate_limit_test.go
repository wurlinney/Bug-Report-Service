package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRateLimit_AllowsWithinBurstThenBlocks(t *testing.T) {
	h := RateLimit(1, 2)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// First two should pass due to burst=2
	for i := 0; i < 2; i++ {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "http://example.test/", nil)
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200 on request %d, got %d", i+1, rr.Code)
		}
	}

	// Third is likely to be rate-limited (no time has passed)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://example.test/", nil)
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", rr.Code)
	}
}
