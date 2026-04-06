package change_report_meta

import (
	"context"

	uc "bug-report-service/internal/usecase/change_report_meta"
)

type UseCase interface {
	Execute(ctx context.Context, req uc.Request) error
}
