package delete_upload

import (
	"context"
	"strings"

	"bug-report-service/internal/domain"
)

type UseCase struct {
	attachments AttachmentDeleter
}

func New(attachments AttachmentDeleter) *UseCase {
	return &UseCase{attachments: attachments}
}

func (uc *UseCase) Execute(ctx context.Context, uploadSessionID string, storageKey string) (bool, error) {
	if strings.TrimSpace(uploadSessionID) == "" || strings.TrimSpace(storageKey) == "" {
		return false, domain.ErrBadInput
	}
	return uc.attachments.DeleteFromSessionByStorageKey(ctx, uploadSessionID, storageKey)
}
