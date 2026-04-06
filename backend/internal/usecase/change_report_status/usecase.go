package change_report_status

import (
	"context"
	"errors"

	"bug-report-service/internal/domain"
	"bug-report-service/internal/domain/policy"
	"bug-report-service/internal/domain/report"
)

type Request struct {
	ActorRole string
	ReportID  string
	Status    string
}

type UseCase struct {
	reports ReportStatusUpdater
}

func New(reports ReportStatusUpdater) *UseCase {
	return &UseCase{reports: reports}
}

func (uc *UseCase) Execute(ctx context.Context, req Request) error {
	if !policy.CanModeratorChangeStatus(req.ActorRole) {
		return domain.ErrForbidden
	}
	if req.ReportID == "" || !report.IsValidStatus(req.Status) {
		return domain.ErrBadInput
	}
	if err := uc.reports.UpdateStatus(ctx, req.ReportID, req.Status); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.ErrNotFound
		}
		return err
	}
	return nil
}
