package change_report_meta

import (
	"context"
	"errors"
	"testing"

	"bug-report-service/internal/domain"
	"bug-report-service/internal/domain/report"
)

type mockReportMetaUpdater struct {
	err error
}

func (m *mockReportMetaUpdater) UpdateMeta(_ context.Context, _ string, _ string, _ string) error {
	return m.err
}

func TestExecute_Success(t *testing.T) {
	uc := New(&mockReportMetaUpdater{})

	err := uc.Execute(context.Background(), Request{
		ActorRole: "moderator",
		ReportID:  "r1",
		Priority:  report.PriorityHigh,
		Influence: report.InfluenceMedium,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExecute_Error_NonModerator(t *testing.T) {
	uc := New(&mockReportMetaUpdater{})

	err := uc.Execute(context.Background(), Request{
		ActorRole: "user",
		ReportID:  "r1",
		Priority:  report.PriorityHigh,
		Influence: report.InfluenceMedium,
	})
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestExecute_Error_InvalidPriority(t *testing.T) {
	uc := New(&mockReportMetaUpdater{})

	err := uc.Execute(context.Background(), Request{
		ActorRole: "moderator",
		ReportID:  "r1",
		Priority:  "bogus",
		Influence: report.InfluenceMedium,
	})
	if !errors.Is(err, domain.ErrBadInput) {
		t.Fatalf("expected ErrBadInput, got %v", err)
	}
}
