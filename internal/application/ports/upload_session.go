package ports

import (
	"context"
	"time"
)

type UploadSessionRepository interface {
	Create(ctx context.Context) (UploadSessionRecord, error)
	GetByID(ctx context.Context, id string) (UploadSessionRecord, bool, error)
}

type UploadSessionRecord struct {
	ID        string
	CreatedAt time.Time
}
