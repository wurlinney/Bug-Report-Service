package change_report_status

import (
	"context"

	ucMeta "bug-report-service/internal/usecase/change_report_meta"
	ucStatus "bug-report-service/internal/usecase/change_report_status"
)

type StatusUseCase interface {
	Execute(ctx context.Context, req ucStatus.Request) error
}

type MetaUseCase interface {
	Execute(ctx context.Context, req ucMeta.Request) error
}
