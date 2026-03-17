package httpserver

import (
	"net/http"
	"testing"
)

func TestReadiness_DefaultNotReady(t *testing.T) {
	r := NewReadiness()
	code, _ := r.ReadyResponse()
	if code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", code)
	}
}

func TestReadiness_WhenAllDepsReady(t *testing.T) {
	r := NewReadiness()
	r.SetDependency("db", true)
	r.SetDependency("s3", true)

	code, _ := r.ReadyResponse()
	if code != http.StatusOK {
		t.Fatalf("expected 200, got %d", code)
	}
}

func TestReadiness_ShuttingDown(t *testing.T) {
	r := NewReadiness()
	r.SetDependency("db", true)
	r.SetDependency("s3", true)
	r.SetShuttingDown()

	code, _ := r.ReadyResponse()
	if code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", code)
	}
}
