package change_report_meta

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
	Priority  string
	Influence string
}

type UseCase struct {
	reports ReportMetaUpdater
}

func New(reports ReportMetaUpdater) *UseCase {
	return &UseCase{reports: reports}
}

func (uc *UseCase) Execute(ctx context.Context, req Request) error {
	if !policy.CanModeratorChangeStatus(req.ActorRole) {
		return domain.ErrForbidden
	}
	if req.ReportID == "" || !report.IsValidPriority(req.Priority) || !report.IsValidInfluence(req.Influence) {
		return domain.ErrBadInput
	}
	if err := uc.reports.UpdateMeta(ctx, req.ReportID, req.Priority, req.Influence); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.ErrNotFound
		}
		return err
	}
	return nil
}
