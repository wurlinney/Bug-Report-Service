package list_reports

import (
	"context"

	"bug-report-service/internal/domain/report"
	uc "bug-report-service/internal/usecase/list_reports"
)

type UseCase interface {
	Execute(ctx context.Context, req uc.Request) ([]report.Report, int, error)
}
