package note

import (
	"context"
	"strings"

	"bug-report-service/internal/application/policy"
	"bug-report-service/internal/application/ports"
)

type Deps struct {
	Notes   ports.InternalNoteRepository
	Reports ports.ReportRepository
}

type Service struct {
	deps Deps
}

func NewService(deps Deps) *Service { return &Service{deps: deps} }

type CreateRequest struct {
	ActorRole string
	ActorID   string
	ReportID  string
	Text      string
}

type ListRequest struct {
	ActorRole string
	ReportID  string
	Limit     int
	Offset    int
}

type NoteDTO struct {
	ID                string
	ReportID          string
	AuthorModeratorID string
	Text              string
	CreatedAt         int64
}

func (s *Service) Create(ctx context.Context, req CreateRequest) (NoteDTO, error) {
	if !policy.CanModeratorChangeStatus(req.ActorRole) {
		return NoteDTO{}, ErrForbidden
	}
	if strings.TrimSpace(req.ActorID) == "" || strings.TrimSpace(req.ReportID) == "" || strings.TrimSpace(req.Text) == "" {
		return NoteDTO{}, ErrBadInput
	}
	_, found, err := s.deps.Reports.GetByID(ctx, req.ReportID)
	if err != nil {
		return NoteDTO{}, err
	}
	if !found {
		return NoteDTO{}, ErrNotFound
	}

	rec := ports.InternalNoteRecord{
		ReportID:          strings.TrimSpace(req.ReportID),
		AuthorModeratorID: strings.TrimSpace(req.ActorID),
		Text:              strings.TrimSpace(req.Text),
	}
	created, err := s.deps.Notes.Create(ctx, rec)
	if err != nil {
		return NoteDTO{}, err
	}
	return toDTO(created), nil
}

func (s *Service) List(ctx context.Context, req ListRequest) ([]NoteDTO, int, error) {
	if !policy.CanModeratorChangeStatus(req.ActorRole) {
		return nil, 0, ErrForbidden
	}
	if strings.TrimSpace(req.ReportID) == "" {
		return nil, 0, ErrBadInput
	}
	_, found, err := s.deps.Reports.GetByID(ctx, req.ReportID)
	if err != nil {
		return nil, 0, err
	}
	if !found {
		return nil, 0, ErrNotFound
	}

	limit := req.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset := req.Offset
	if offset < 0 {
		offset = 0
	}

	items, total, err := s.deps.Notes.ListByReport(ctx, req.ReportID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	out := make([]NoteDTO, 0, len(items))
	for _, it := range items {
		out = append(out, toDTO(it))
	}
	return out, total, nil
}

func toDTO(n ports.InternalNoteRecord) NoteDTO {
	return NoteDTO{
		ID:                n.ID,
		ReportID:          n.ReportID,
		AuthorModeratorID: n.AuthorModeratorID,
		Text:              n.Text,
		CreatedAt:         n.CreatedAt.Unix(),
	}
}
