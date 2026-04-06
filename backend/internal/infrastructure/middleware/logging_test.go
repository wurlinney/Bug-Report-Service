package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"bug-report-service/internal/infrastructure/logger"
)

// spyLogger implements logger.Logger and records Info calls for inspection.
type spyLogger struct {
	infoCalls  []spyLogCall
	errorCalls []spyLogCall
}

type spyLogCall struct {
	msg    string
	fields []any
}

func (s *spyLogger) Info(msg string, fields ...any) {
	s.infoCalls = append(s.infoCalls, spyLogCall{msg: msg, fields: fields})
}

func (s *spyLogger) Error(msg string, fields ...any) {
	s.errorCalls = append(s.errorCalls, spyLogCall{msg: msg, fields: fields})
}

func (s *spyLogger) With(fields ...any) logger.Logger { return s }
func (s *spyLogger) Sync() error                      { return nil }

// fieldValue extracts a field value by key from a flat key-value fields slice.
func fieldValue(fields []any, key string) any {
	for i := 0; i+1 < len(fields); i += 2 {
		if k, ok := fields[i].(string); ok && k == key {
			return fields[i+1]
		}
	}
	return nil
}

func TestLogging_LogsAfterHandlerCompletes(t *testing.T) {
	log := &spyLogger{}
	handler := Logging(log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("hello"))
	}))

	req := httptest.NewRequest(http.MethodPost, "/reports", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if len(log.infoCalls) != 1 {
		t.Fatalf("expected 1 info log call, got %d", len(log.infoCalls))
	}
	call := log.infoCalls[0]
	if call.msg != "http request" {
		t.Fatalf("log message: got %q, want %q", call.msg, "http request")
	}
}

func TestLogging_RecordsCorrectStatusCode(t *testing.T) {
	log := &spyLogger{}
	handler := Logging(log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))

	req := httptest.NewRequest(http.MethodGet, "/missing", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if len(log.infoCalls) == 0 {
		t.Fatal("expected at least 1 info log call")
	}
	status := fieldValue(log.infoCalls[0].fields, "status")
	if status != 404 {
		t.Fatalf("logged status: got %v, want 404", status)
	}
}

func TestLogging_RecordsCorrectByteCount(t *testing.T) {
	log := &spyLogger{}
	body := []byte("response body content")
	handler := Logging(log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(body)
	}))

	req := httptest.NewRequest(http.MethodGet, "/data", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if len(log.infoCalls) == 0 {
		t.Fatal("expected at least 1 info log call")
	}
	bytes := fieldValue(log.infoCalls[0].fields, "bytes")
	if bytes != len(body) {
		t.Fatalf("logged bytes: got %v, want %d", bytes, len(body))
	}
}
