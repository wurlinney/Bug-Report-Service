package get_report

import (
	"context"

	"bug-report-service/internal/domain/report"
)

type UseCase interface {
	Execute(ctx context.Context, actorRole string, reportID string) (report.Report, error)
}
