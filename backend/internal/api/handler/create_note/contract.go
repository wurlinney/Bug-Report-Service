package create_note

import (
	"context"

	"bug-report-service/internal/domain/note"
	uc "bug-report-service/internal/usecase/create_note"
)

type UseCase interface {
	Execute(ctx context.Context, req uc.Request) (note.Note, error)
}
