package create_report

import (
	"context"

	"bug-report-service/internal/domain/report"
	uc "bug-report-service/internal/usecase/create_report"
)

type UseCase interface {
	Execute(ctx context.Context, req uc.Request) (report.Report, error)
}
