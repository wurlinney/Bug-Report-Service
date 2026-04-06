package create_note

import (
	"context"

	"bug-report-service/internal/domain/note"
	"bug-report-service/internal/domain/report"
)

type NoteCreator interface {
	Create(ctx context.Context, n note.Note) (note.Note, error)
}

type ReportGetter interface {
	GetByID(ctx context.Context, id string) (report.Report, bool, error)
}
