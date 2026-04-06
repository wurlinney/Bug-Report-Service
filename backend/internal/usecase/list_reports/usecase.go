package list_reports

import (
	"context"
	"time"

	"bug-report-service/internal/domain"
	"bug-report-service/internal/domain/policy"
	"bug-report-service/internal/domain/report"
)

type Request struct {
	ActorRole string

	Status       *string
	ReporterName *string
	Query        *string
	CreatedFrom  *time.Time
	CreatedTo    *time.Time

	SortBy   string
	SortDesc bool
	Limit    int
	Offset   int
}

type UseCase struct {
	reports ReportLister
}

func New(reports ReportLister) *UseCase {
	return &UseCase{reports: reports}
}

func (uc *UseCase) Execute(ctx context.Context, req Request) ([]report.Report, int, error) {
	if !policy.CanModeratorChangeStatus(req.ActorRole) {
		return nil, 0, domain.ErrForbidden
	}
	f := report.ListFilter{
		Status:       req.Status,
		ReporterName: req.ReporterName,
		Query:        req.Query,
		SortBy:       req.SortBy,
		SortDesc:     req.SortDesc,
		Limit:        req.Limit,
		Offset:       req.Offset,
	}
	if req.CreatedFrom != nil || req.CreatedTo != nil {
		f.CreatedAt = &report.TimeRange{From: req.CreatedFrom, To: req.CreatedTo}
	}
	return uc.reports.ListAll(ctx, f)
}
