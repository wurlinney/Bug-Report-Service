package finalize_attachment

import (
	"context"
	"path/filepath"
	"regexp"
	"strings"

	"bug-report-service/internal/domain"
	"bug-report-service/internal/domain/attachment"
)

var reUnsafe = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

type Request struct {
	UploadSessionID string
	UploadID        string
	FileName        string
	ContentType     string
	FileSize        int64
	StorageKey      string
	IdempotencyKey  string
}

type UseCase struct {
	sessions     SessionGetter
	attachments  AttachmentCreator
	idempotency  IdempotencyChecker
	maxFileSize  int64
	allowedMIMEs map[string]struct{}
}

func New(
	sessions SessionGetter,
	attachments AttachmentCreator,
	idempotency IdempotencyChecker,
	maxFileSize int64,
	allowedMIMEs map[string]struct{},
) *UseCase {
	return &UseCase{
		sessions:     sessions,
		attachments:  attachments,
		idempotency:  idempotency,
		maxFileSize:  maxFileSize,
		allowedMIMEs: allowedMIMEs,
	}
}

func (uc *UseCase) Execute(ctx context.Context, req Request) (attachment.Attachment, error) {
	if req.UploadSessionID == "" {
		return attachment.Attachment{}, domain.ErrBadInput
	}
	if strings.TrimSpace(req.UploadID) == "" {
		return attachment.Attachment{}, domain.ErrBadInput
	}
	if req.ContentType == "" || req.FileSize <= 0 || strings.TrimSpace(req.StorageKey) == "" {
		return attachment.Attachment{}, domain.ErrBadInput
	}
	if uc.maxFileSize > 0 && req.FileSize > uc.maxFileSize {
		return attachment.Attachment{}, domain.ErrBadInput
	}
	if _, ok := uc.allowedMIMEs[req.ContentType]; !ok {
		return attachment.Attachment{}, domain.ErrBadInput
	}

	_, found, err := uc.sessions.GetByID(ctx, req.UploadSessionID)
	if err != nil {
		return attachment.Attachment{}, err
	}
	if !found {
		return attachment.Attachment{}, domain.ErrNotFound
	}
	if req.IdempotencyKey != "" {
		existing, ok, err := uc.idempotency.GetByIdempotencyKey(ctx, "", req.UploadSessionID, req.IdempotencyKey)
		if err != nil {
			return attachment.Attachment{}, err
		}
		if ok {
			return existing, nil
		}
	}

	rec := attachment.Attachment{
		UploadSessionID: req.UploadSessionID,
		FileName:        safeFileName(req.FileName),
		ContentType:     req.ContentType,
		FileSize:        req.FileSize,
		StorageKey:      strings.TrimSpace(req.StorageKey),
		IdempotencyKey:  req.IdempotencyKey,
	}
	return uc.attachments.Create(ctx, rec)
}

func safeFileName(name string) string {
	base := filepath.Base(strings.TrimSpace(name))
	if base == "." || base == string(filepath.Separator) || base == "" {
		return "file"
	}
	base = reUnsafe.ReplaceAllString(base, "_")
	base = strings.Trim(base, "._-")
	if base == "" {
		base = "file"
	}
	if len(base) > 128 {
		base = base[:128]
	}
	return base
}
