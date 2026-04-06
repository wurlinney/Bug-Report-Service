package change_report_status

import (
	"context"
	"errors"
	"testing"

	"bug-report-service/internal/domain"
	"bug-report-service/internal/domain/report"
)

type mockReportStatusUpdater struct {
	err error
}

func (m *mockReportStatusUpdater) UpdateStatus(_ context.Context, _ string, _ string) error {
	return m.err
}

func TestExecute_Success(t *testing.T) {
	uc := New(&mockReportStatusUpdater{})

	err := uc.Execute(context.Background(), Request{
		ActorRole: "moderator",
		ReportID:  "r1",
		Status:    report.StatusInReview,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExecute_Error_NonModerator(t *testing.T) {
	uc := New(&mockReportStatusUpdater{})

	err := uc.Execute(context.Background(), Request{
		ActorRole: "user",
		ReportID:  "r1",
		Status:    report.StatusInReview,
	})
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestExecute_Error_InvalidStatus(t *testing.T) {
	uc := New(&mockReportStatusUpdater{})

	err := uc.Execute(context.Background(), Request{
		ActorRole: "moderator",
		ReportID:  "r1",
		Status:    "bogus",
	})
	if !errors.Is(err, domain.ErrBadInput) {
		t.Fatalf("expected ErrBadInput, got %v", err)
	}
}

func TestExecute_Error_ReportNotFound(t *testing.T) {
	uc := New(&mockReportStatusUpdater{err: domain.ErrNotFound})

	err := uc.Execute(context.Background(), Request{
		ActorRole: "moderator",
		ReportID:  "r1",
		Status:    report.StatusResolved,
	})
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
