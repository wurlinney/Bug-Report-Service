package create_note

import (
	"context"
	"strings"

	"bug-report-service/internal/domain"
	"bug-report-service/internal/domain/note"
	"bug-report-service/internal/domain/policy"
)

type Request struct {
	ActorRole string
	ActorID   string
	ReportID  string
	Text      string
}

type UseCase struct {
	notes   NoteCreator
	reports ReportGetter
}

func New(notes NoteCreator, reports ReportGetter) *UseCase {
	return &UseCase{notes: notes, reports: reports}
}

func (uc *UseCase) Execute(ctx context.Context, req Request) (note.Note, error) {
	if !policy.CanModeratorChangeStatus(req.ActorRole) {
		return note.Note{}, domain.ErrForbidden
	}
	if strings.TrimSpace(req.ActorID) == "" || strings.TrimSpace(req.ReportID) == "" || strings.TrimSpace(req.Text) == "" {
		return note.Note{}, domain.ErrBadInput
	}
	_, found, err := uc.reports.GetByID(ctx, req.ReportID)
	if err != nil {
		return note.Note{}, err
	}
	if !found {
		return note.Note{}, domain.ErrNotFound
	}

	rec := note.Note{
		ReportID:          strings.TrimSpace(req.ReportID),
		AuthorModeratorID: strings.TrimSpace(req.ActorID),
		Text:              strings.TrimSpace(req.Text),
	}
	return uc.notes.Create(ctx, rec)
}
