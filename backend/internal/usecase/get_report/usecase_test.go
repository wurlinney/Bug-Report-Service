package get_report

import (
	"context"
	"errors"
	"testing"

	"bug-report-service/internal/domain"
	"bug-report-service/internal/domain/report"
)

type mockReportGetter struct {
	report report.Report
	found  bool
	err    error
}

func (m *mockReportGetter) GetByID(_ context.Context, _ string) (report.Report, bool, error) {
	return m.report, m.found, m.err
}

func TestExecute_Success_Moderator(t *testing.T) {
	uc := New(&mockReportGetter{
		report: report.Report{ID: "r1", ReporterName: "Alice"},
		found:  true,
	})

	got, err := uc.Execute(context.Background(), "moderator", "r1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != "r1" {
		t.Errorf("expected ID r1, got %s", got.ID)
	}
}

func TestExecute_Error_NotFound(t *testing.T) {
	uc := New(&mockReportGetter{found: false})

	_, err := uc.Execute(context.Background(), "moderator", "missing")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestExecute_Error_NonModerator(t *testing.T) {
	uc := New(&mockReportGetter{
		report: report.Report{ID: "r1"},
		found:  true,
	})

	_, err := uc.Execute(context.Background(), "user", "r1")
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}
