package ports

import (
	"context"
	"time"
)

type AttachmentRepository interface {
	Create(ctx context.Context, a AttachmentRecord) error
	GetByIdempotencyKey(ctx context.Context, reportID string, key string) (a AttachmentRecord, found bool, err error)
	ListByReport(ctx context.Context, reportID string) ([]AttachmentRecord, error)
	ExistsByStorageKey(ctx context.Context, storageKey string) (bool, error)
}

type AttachmentRecord struct {
	ID             string
	ReportID       string
	FileName       string
	ContentType    string
	FileSize       int64
	StorageKey     string
	CreatedAt      time.Time
	IdempotencyKey string
}

type ObjectStorage interface {
	PutObject(ctx context.Context, key string, contentType string, data []byte) error
	DeleteObject(ctx context.Context, key string) error
}

type ObjectURLSigner interface {
	PresignGetObject(ctx context.Context, key string, expiresIn time.Duration) (string, error)
}
