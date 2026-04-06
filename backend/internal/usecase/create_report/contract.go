package create_report

import (
	"context"

	"bug-report-service/internal/domain/report"
)

type ReportCreator interface {
	Create(ctx context.Context, r report.Report) (report.Report, error)
}

type SessionGetter interface {
	GetByID(ctx context.Context, id string) (found bool, err error)
}

type AttachmentBinder interface {
	BindSessionToReport(ctx context.Context, uploadSessionID string, reportID string) error
}
