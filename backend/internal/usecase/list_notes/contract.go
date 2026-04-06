package list_notes

import (
	"context"

	"bug-report-service/internal/domain/note"
	"bug-report-service/internal/domain/report"
)

type NoteLister interface {
	ListByReport(ctx context.Context, reportID string, limit int, offset int) ([]note.Note, int, error)
}

type ReportGetter interface {
	GetByID(ctx context.Context, id string) (report.Report, bool, error)
}
