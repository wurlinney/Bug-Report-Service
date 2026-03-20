package ports

import (
	"context"
	"time"
)

type AttachmentRepository interface {
	Create(ctx context.Context, a AttachmentRecord) (AttachmentRecord, error)
	GetByIdempotencyKey(ctx context.Context, reportID string, uploadSessionID string, key string) (a AttachmentRecord, found bool, err error)
	ListByReport(ctx context.Context, reportID string) ([]AttachmentRecord, error)
	ExistsByStorageKey(ctx context.Context, storageKey string) (bool, error)
	BindSessionToReport(ctx context.Context, uploadSessionID string, reportID string) error
	DeleteFromSessionByStorageKey(ctx context.Context, uploadSessionID string, storageKey string) (deleted bool, err error)
}

type AttachmentRecord struct {
	ID              int64
	ReportID        string
	UploadSessionID string
	FileName        string
	ContentType     string
	FileSize        int64
	StorageKey      string
	CreatedAt       time.Time
	IdempotencyKey  string
}

type ObjectStorage interface {
	PutObject(ctx context.Context, key string, contentType string, data []byte) error
	DeleteObject(ctx context.Context, key string) error
}

type ObjectURLSigner interface {
	PresignGetObject(ctx context.Context, key string, expiresIn time.Duration) (string, error)
}
