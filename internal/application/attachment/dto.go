package attachment

import "time"

type UploadRequest struct {
	ReportID string

	FileName    string
	ContentType string
	Data        []byte

	IdempotencyKey string
}

// FinalizeRequest is used when the file bytes are uploaded externally (e.g. tusd),
// and we only need to validate access and persist metadata in DB.
type FinalizeRequest struct {
	ReportID string

	UploadID       string
	FileName       string
	ContentType    string
	FileSize       int64
	StorageKey     string
	IdempotencyKey string
}

type DTO struct {
	ID          string
	ReportID    string
	FileName    string
	ContentType string
	FileSize    int64
	StorageKey  string
	CreatedAt   time.Time
}

type ListForReportRequest struct {
	ActorRole string
	ActorID   string
	ReportID  string
}
