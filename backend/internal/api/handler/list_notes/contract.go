package list_notes

import (
	"context"

	"bug-report-service/internal/domain/note"
	uc "bug-report-service/internal/usecase/list_notes"
)

type UseCase interface {
	Execute(ctx context.Context, req uc.Request) ([]note.Note, int, error)
}
