package list_notes

import (
	"context"
	"errors"
	"testing"

	"bug-report-service/internal/domain"
	"bug-report-service/internal/domain/note"
	"bug-report-service/internal/domain/report"
)

type mockNoteLister struct {
	notes []note.Note
	total int
	err   error
}

func (m *mockNoteLister) ListByReport(_ context.Context, _ string, _ int, _ int) ([]note.Note, int, error) {
	return m.notes, m.total, m.err
}

type mockReportGetter struct {
	report report.Report
	found  bool
	err    error
}

func (m *mockReportGetter) GetByID(_ context.Context, _ string) (report.Report, bool, error) {
	return m.report, m.found, m.err
}

func TestExecute_Success(t *testing.T) {
	uc := New(
		&mockNoteLister{
			notes: []note.Note{{ID: "n1"}, {ID: "n2"}},
			total: 2,
		},
		&mockReportGetter{found: true},
	)

	notes, total, err := uc.Execute(context.Background(), Request{
		ActorRole: "moderator",
		ReportID:  "r1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(notes) != 2 {
		t.Errorf("expected 2 notes, got %d", len(notes))
	}
	if total != 2 {
		t.Errorf("expected total 2, got %d", total)
	}
}

func TestExecute_Error_NonModerator(t *testing.T) {
	uc := New(&mockNoteLister{}, &mockReportGetter{found: true})

	_, _, err := uc.Execute(context.Background(), Request{
		ActorRole: "user",
		ReportID:  "r1",
	})
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestExecute_Error_ReportNotFound(t *testing.T) {
	uc := New(&mockNoteLister{}, &mockReportGetter{found: false})

	_, _, err := uc.Execute(context.Background(), Request{
		ActorRole: "moderator",
		ReportID:  "r1",
	})
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
