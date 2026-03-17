package message

import (
	"context"
	"strings"

	"bug-report-service/internal/application/policy"
	"bug-report-service/internal/application/ports"
)

type Deps struct {
	Reports  ports.ReportRepository
	Messages ports.MessageRepository
	Clock    ports.Clock
	Random   ports.Random
}

type Service struct {
	deps Deps
}

func NewService(deps Deps) *Service {
	return &Service{deps: deps}
}

func (s *Service) Create(ctx context.Context, req CreateRequest) (MessageDTO, error) {
	text := strings.TrimSpace(req.Text)
	if req.ActorRole == "" || req.ActorID == "" || req.ReportID == "" || text == "" {
		return MessageDTO{}, ErrBadInput
	}

	rep, found, err := s.deps.Reports.GetByID(ctx, req.ReportID)
	if err != nil {
		return MessageDTO{}, err
	}
	if !found {
		return MessageDTO{}, ErrNotFound
	}

	// moderator can post to any report; user only to own report
	if !policy.CanUserViewReport(req.ActorRole, req.ActorID, rep.UserID) {
		return MessageDTO{}, ErrForbidden
	}

	now := s.deps.Clock.Now()
	id := s.deps.Random.NewID()
	rec := ports.MessageRecord{
		ID:         id,
		ReportID:   req.ReportID,
		SenderID:   req.ActorID,
		SenderRole: req.ActorRole,
		Text:       text,
		CreatedAt:  now,
	}
	if err := s.deps.Messages.Create(ctx, rec); err != nil {
		return MessageDTO{}, err
	}

	return toDTO(rec), nil
}

func (s *Service) List(ctx context.Context, req ListRequest) (ListResponse, error) {
	if req.ActorRole == "" || req.ActorID == "" || req.ReportID == "" {
		return ListResponse{}, ErrBadInput
	}

	rep, found, err := s.deps.Reports.GetByID(ctx, req.ReportID)
	if err != nil {
		return ListResponse{}, err
	}
	if !found {
		return ListResponse{}, ErrNotFound
	}

	if !policy.CanUserViewReport(req.ActorRole, req.ActorID, rep.UserID) {
		return ListResponse{}, ErrForbidden
	}

	items, total, err := s.deps.Messages.ListByReport(ctx, req.ReportID, ports.MessageListFilter{
		Limit:    req.Limit,
		Offset:   req.Offset,
		SortDesc: req.SortDesc,
	})
	if err != nil {
		return ListResponse{}, err
	}

	out := make([]MessageDTO, 0, len(items))
	for _, it := range items {
		out = append(out, toDTO(it))
	}
	return ListResponse{Items: out, Total: total}, nil
}

func toDTO(m ports.MessageRecord) MessageDTO {
	return MessageDTO{
		ID:         m.ID,
		ReportID:   m.ReportID,
		SenderID:   m.SenderID,
		SenderRole: m.SenderRole,
		Text:       m.Text,
		CreatedAt:  m.CreatedAt,
	}
}
