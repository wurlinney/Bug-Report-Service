package report

import (
	"context"
	"errors"
	"strings"

	"bug-report-service/internal/application/policy"
	"bug-report-service/internal/application/ports"
)

type Deps struct {
	Reports ports.ReportRepository
	Clock   ports.Clock
	Random  ports.Random
}

type Service struct {
	deps Deps
}

func NewService(deps Deps) *Service {
	return &Service{deps: deps}
}

func (s *Service) Exists(ctx context.Context, reportID string) (bool, error) {
	if strings.TrimSpace(reportID) == "" {
		return false, ErrBadInput
	}
	_, found, err := s.deps.Reports.GetByID(ctx, reportID)
	if err != nil {
		return false, err
	}
	return found, nil
}

func (s *Service) Create(ctx context.Context, req CreateRequest) (ReportDTO, error) {
	reporterName := strings.TrimSpace(req.ReporterName)
	desc := strings.TrimSpace(req.Description)
	if reporterName == "" {
		return ReportDTO{}, ErrBadInput
	}

	now := s.deps.Clock.Now()
	id := s.deps.Random.NewID()
	r := ports.ReportRecord{
		ID:           id,
		ReporterName: reporterName,
		Description:  desc,
		Status:       StatusNew,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := s.deps.Reports.Create(ctx, r); err != nil {
		return ReportDTO{}, err
	}
	return toDTO(r), nil
}

func (s *Service) GetForActor(ctx context.Context, actorRole string, _ string, reportID string) (ReportDTO, error) {
	r, found, err := s.deps.Reports.GetByID(ctx, reportID)
	if err != nil {
		return ReportDTO{}, err
	}
	if !found {
		return ReportDTO{}, ErrNotFound
	}
	if !policy.CanModeratorChangeStatus(actorRole) {
		return ReportDTO{}, ErrForbidden
	}
	return toDTO(r), nil
}

func (s *Service) ChangeStatus(ctx context.Context, req ChangeStatusRequest) error {
	if !policy.CanModeratorChangeStatus(req.ActorRole) {
		return ErrForbidden
	}
	if req.ReportID == "" || !IsValidStatus(req.Status) {
		return ErrBadInput
	}
	now := s.deps.Clock.Now()
	if err := s.deps.Reports.UpdateStatus(ctx, req.ReportID, req.Status, now); err != nil {
		if errors.Is(err, ports.ErrNotFound) {
			return ErrNotFound
		}
		return err
	}
	return nil
}

func (s *Service) ListAll(ctx context.Context, req ListAllRequest) ([]ReportDTO, int, error) {
	if !policy.CanModeratorChangeStatus(req.ActorRole) {
		return nil, 0, ErrForbidden
	}
	f := ports.ReportListFilter{
		Status:       req.Status,
		ReporterName: req.ReporterName,
		Query:        req.Query,
		SortBy:       req.SortBy,
		SortDesc:     req.SortDesc,
		Limit:        req.Limit,
		Offset:       req.Offset,
	}
	if req.CreatedFrom != nil || req.CreatedTo != nil {
		f.CreatedAt = &ports.TimeRange{From: req.CreatedFrom, To: req.CreatedTo}
	}
	items, total, err := s.deps.Reports.ListAll(ctx, f)
	if err != nil {
		return nil, 0, err
	}
	out := make([]ReportDTO, 0, len(items))
	for _, r := range items {
		out = append(out, toDTO(r))
	}
	return out, total, nil
}

func toDTO(r ports.ReportRecord) ReportDTO {
	return ReportDTO{
		ID:           r.ID,
		ReporterName: r.ReporterName,
		Description:  r.Description,
		Status:       r.Status,
		CreatedAt:    r.CreatedAt,
		UpdatedAt:    r.UpdatedAt,
	}
}
