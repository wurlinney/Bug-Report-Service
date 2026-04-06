package list_notes

import (
	"context"
	"strings"

	"bug-report-service/internal/domain"
	"bug-report-service/internal/domain/note"
	"bug-report-service/internal/domain/policy"
)

type Request struct {
	ActorRole string
	ReportID  string
	Limit     int
	Offset    int
}

type UseCase struct {
	notes   NoteLister
	reports ReportGetter
}

func New(notes NoteLister, reports ReportGetter) *UseCase {
	return &UseCase{notes: notes, reports: reports}
}

func (uc *UseCase) Execute(ctx context.Context, req Request) ([]note.Note, int, error) {
	if !policy.CanModeratorChangeStatus(req.ActorRole) {
		return nil, 0, domain.ErrForbidden
	}
	if strings.TrimSpace(req.ReportID) == "" {
		return nil, 0, domain.ErrBadInput
	}
	_, found, err := uc.reports.GetByID(ctx, req.ReportID)
	if err != nil {
		return nil, 0, err
	}
	if !found {
		return nil, 0, domain.ErrNotFound
	}

	limit := req.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset := req.Offset
	if offset < 0 {
		offset = 0
	}

	return uc.notes.ListByReport(ctx, req.ReportID, limit, offset)
}
