package ports

import (
	"context"
	"time"
)

type InternalNoteRepository interface {
	Create(ctx context.Context, n InternalNoteRecord) (InternalNoteRecord, error)
	ListByReport(ctx context.Context, reportID string, limit int, offset int) ([]InternalNoteRecord, int, error)
}

type InternalNoteRecord struct {
	ID                string
	ReportID          string
	AuthorModeratorID string
	Text              string
	CreatedAt         time.Time
}
