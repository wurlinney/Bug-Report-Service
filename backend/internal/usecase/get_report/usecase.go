package get_report

import (
	"context"

	"bug-report-service/internal/domain"
	"bug-report-service/internal/domain/policy"
	"bug-report-service/internal/domain/report"
)

type UseCase struct {
	reports ReportGetter
}

func New(reports ReportGetter) *UseCase {
	return &UseCase{reports: reports}
}

func (uc *UseCase) Execute(ctx context.Context, actorRole string, reportID string) (report.Report, error) {
	r, found, err := uc.reports.GetByID(ctx, reportID)
	if err != nil {
		return report.Report{}, err
	}
	if !found {
		return report.Report{}, domain.ErrNotFound
	}
	if !policy.CanModeratorChangeStatus(actorRole) {
		return report.Report{}, domain.ErrForbidden
	}
	return r, nil
}
