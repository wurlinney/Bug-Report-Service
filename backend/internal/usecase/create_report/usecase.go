package create_report

import (
	"context"
	"strings"

	"bug-report-service/internal/domain"
	"bug-report-service/internal/domain/report"
)

type Request struct {
	ReporterName    string
	Description     string
	UploadSessionID string
}

type UseCase struct {
	reports     ReportCreator
	sessions    SessionGetter
	attachments AttachmentBinder
}

func New(reports ReportCreator, sessions SessionGetter, attachments AttachmentBinder) *UseCase {
	return &UseCase{reports: reports, sessions: sessions, attachments: attachments}
}

func (uc *UseCase) Execute(ctx context.Context, req Request) (report.Report, error) {
	reporterName := strings.TrimSpace(req.ReporterName)
	desc := strings.TrimSpace(req.Description)
	if reporterName == "" {
		return report.Report{}, domain.ErrBadInput
	}

	r := report.Report{
		ReporterName: reporterName,
		Description:  desc,
		Status:       report.StatusNew,
	}
	created, err := uc.reports.Create(ctx, r)
	if err != nil {
		return report.Report{}, err
	}
	if uploadSessionID := strings.TrimSpace(req.UploadSessionID); uploadSessionID != "" {
		if uc.sessions == nil || uc.attachments == nil {
			return report.Report{}, domain.ErrBadInput
		}
		found, err := uc.sessions.GetByID(ctx, uploadSessionID)
		if err != nil {
			return report.Report{}, err
		}
		if !found {
			return report.Report{}, domain.ErrBadInput
		}
		if err := uc.attachments.BindSessionToReport(ctx, uploadSessionID, created.ID); err != nil {
			return report.Report{}, err
		}
	}
	return created, nil
}
