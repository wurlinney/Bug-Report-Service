package list_attachments

import (
	"context"
	"time"

	"bug-report-service/internal/domain/attachment"
	"bug-report-service/internal/domain/report"
)

type ReportGetter interface {
	GetByID(ctx context.Context, id string) (report.Report, bool, error)
}

type AttachmentLister interface {
	ListByReport(ctx context.Context, reportID string) ([]attachment.Attachment, error)
}

type URLSigner interface {
	PresignGetObject(ctx context.Context, key string, expiresIn time.Duration) (string, error)
}
