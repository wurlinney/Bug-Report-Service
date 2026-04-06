package finalize_attachment

import (
	"context"

	"bug-report-service/internal/domain/attachment"
	"bug-report-service/internal/domain/uploadsession"
)

type SessionGetter interface {
	GetByID(ctx context.Context, id string) (uploadsession.UploadSession, bool, error)
}

type AttachmentCreator interface {
	Create(ctx context.Context, a attachment.Attachment) (attachment.Attachment, error)
}

type IdempotencyChecker interface {
	GetByIdempotencyKey(ctx context.Context, reportID string, uploadSessionID string, key string) (attachment.Attachment, bool, error)
}
