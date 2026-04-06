package attachment

import "time"

type Attachment struct {
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
