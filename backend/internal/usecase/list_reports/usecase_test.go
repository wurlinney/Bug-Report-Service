package list_reports

import (
	"context"
	"errors"
	"testing"

	"bug-report-service/internal/domain"
	"bug-report-service/internal/domain/report"
)

type mockReportLister struct {
	items []report.Report
	total int
	err   error
}

func (m *mockReportLister) ListAll(_ context.Context, _ report.ListFilter) ([]report.Report, int, error) {
	return m.items, m.total, m.err
}

func TestExecute_Success_Moderator(t *testing.T) {
	uc := New(&mockReportLister{
		items: []report.Report{{ID: "r1"}, {ID: "r2"}},
		total: 2,
	})

	items, total, err := uc.Execute(context.Background(), Request{ActorRole: "moderator"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 2 {
		t.Errorf("expected 2 items, got %d", len(items))
	}
	if total != 2 {
		t.Errorf("expected total 2, got %d", total)
	}
}

func TestExecute_Error_NonModerator(t *testing.T) {
	uc := New(&mockReportLister{})

	_, _, err := uc.Execute(context.Background(), Request{ActorRole: "user"})
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}
