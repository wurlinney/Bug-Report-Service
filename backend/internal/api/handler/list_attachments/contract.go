package list_attachments

import (
	"context"

	uc "bug-report-service/internal/usecase/list_attachments"
)

type UseCase interface {
	Execute(ctx context.Context, req uc.Request) ([]uc.AttachmentWithURL, error)
}
