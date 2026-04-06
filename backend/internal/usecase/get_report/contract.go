package get_report

import (
	"context"

	"bug-report-service/internal/domain/report"
)

type ReportGetter interface {
	GetByID(ctx context.Context, id string) (report.Report, bool, error)
}
