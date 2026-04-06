package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"bug-report-service/internal/infrastructure/logger"
)

// mockLogger implements logger.Logger for testing.
type mockLogger struct {
	infoCalls  []mockLogCall
	errorCalls []mockLogCall
}

type mockLogCall struct {
	msg    string
	fields []any
}

func (m *mockLogger) Info(msg string, fields ...any) {
	m.infoCalls = append(m.infoCalls, mockLogCall{msg: msg, fields: fields})
}

func (m *mockLogger) Error(msg string, fields ...any) {
	m.errorCalls = append(m.errorCalls, mockLogCall{msg: msg, fields: fields})
}

func (m *mockLogger) With(fields ...any) logger.Logger { return m }
func (m *mockLogger) Sync() error                      { return nil }

func TestRecovery_HandlerPanics(t *testing.T) {
	log := &mockLogger{}
	handler := Recovery(log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("something went wrong")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status: got %d, want 500", rec.Code)
	}

	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode body: %v", err)
	}
	errObj, ok := body["error"].(map[string]any)
	if !ok {
		t.Fatal("expected error object in response body")
	}
	if code, _ := errObj["code"].(string); code != "internal_error" {
		t.Fatalf("error code: got %q, want %q", code, "internal_error")
	}

	if len(log.errorCalls) != 1 {
		t.Fatalf("expected 1 error log call, got %d", len(log.errorCalls))
	}
	if log.errorCalls[0].msg != "panic recovered" {
		t.Fatalf("log message: got %q, want %q", log.errorCalls[0].msg, "panic recovered")
	}
}

func TestRecovery_NoPanic(t *testing.T) {
	log := &mockLogger{}
	handler := Recovery(log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200", rec.Code)
	}
	if rec.Body.String() != "ok" {
		t.Fatalf("body: got %q, want %q", rec.Body.String(), "ok")
	}
	if len(log.errorCalls) != 0 {
		t.Fatalf("expected 0 error log calls, got %d", len(log.errorCalls))
	}
}
