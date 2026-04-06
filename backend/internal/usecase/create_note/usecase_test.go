package create_note

import (
	"context"
	"errors"
	"testing"

	"bug-report-service/internal/domain"
	"bug-report-service/internal/domain/note"
	"bug-report-service/internal/domain/report"
)

type mockNoteCreator struct {
	result note.Note
	err    error
}

func (m *mockNoteCreator) Create(_ context.Context, n note.Note) (note.Note, error) {
	if m.err != nil {
		return note.Note{}, m.err
	}
	out := m.result
	out.ReportID = n.ReportID
	out.AuthorModeratorID = n.AuthorModeratorID
	out.Text = n.Text
	return out, nil
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
		&mockNoteCreator{result: note.Note{ID: "n1"}},
		&mockReportGetter{found: true},
	)

	got, err := uc.Execute(context.Background(), Request{
		ActorRole: "moderator",
		ActorID:   "u1",
		ReportID:  "r1",
		Text:      "Looks like a regression",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != "n1" {
		t.Errorf("expected ID n1, got %s", got.ID)
	}
	if got.Text != "Looks like a regression" {
		t.Errorf("unexpected text: %s", got.Text)
	}
}

func TestExecute_Error_NonModerator(t *testing.T) {
	uc := New(
		&mockNoteCreator{},
		&mockReportGetter{found: true},
	)

	_, err := uc.Execute(context.Background(), Request{
		ActorRole: "user",
		ActorID:   "u1",
		ReportID:  "r1",
		Text:      "note",
	})
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestExecute_Error_EmptyText(t *testing.T) {
	uc := New(
		&mockNoteCreator{},
		&mockReportGetter{found: true},
	)

	_, err := uc.Execute(context.Background(), Request{
		ActorRole: "moderator",
		ActorID:   "u1",
		ReportID:  "r1",
		Text:      "   ",
	})
	if !errors.Is(err, domain.ErrBadInput) {
		t.Fatalf("expected ErrBadInput, got %v", err)
	}
}

func TestExecute_Error_ReportNotFound(t *testing.T) {
	uc := New(
		&mockNoteCreator{},
		&mockReportGetter{found: false},
	)

	_, err := uc.Execute(context.Background(), Request{
		ActorRole: "moderator",
		ActorID:   "u1",
		ReportID:  "missing",
		Text:      "note text",
	})
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
