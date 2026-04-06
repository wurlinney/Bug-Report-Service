package list_reports

import (
	"context"

	"bug-report-service/internal/domain/report"
)

type ReportLister interface {
	ListAll(ctx context.Context, f report.ListFilter) ([]report.Report, int, error)
}
