package attachment

import "time"

type UploadRequest struct {
	ActorRole string
	ActorID   string
	ReportID  string

	FileName    string
	ContentType string
	Data        []byte

	IdempotencyKey string
}

type AttachmentDTO struct {
	ID          string
	ReportID    string
	FileName    string
	ContentType string
	FileSize    int64
	StorageKey  string
	CreatedAt   time.Time
}
