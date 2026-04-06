package list_attachments

import (
	"context"
	"time"

	"bug-report-service/internal/domain"
	"bug-report-service/internal/domain/attachment"
)

type Request struct {
	ActorRole string
	ActorID   string
	ReportID  string
}

type AttachmentWithURL struct {
	attachment.Attachment
	SignedURL string
}

type UseCase struct {
	reports     ReportGetter
	attachments AttachmentLister
	signer      URLSigner
}

func New(reports ReportGetter, attachments AttachmentLister, signer URLSigner) *UseCase {
	return &UseCase{reports: reports, attachments: attachments, signer: signer}
}

func (uc *UseCase) Execute(ctx context.Context, req Request) ([]AttachmentWithURL, error) {
	if req.ActorRole == "" || req.ActorID == "" || req.ReportID == "" {
		return nil, domain.ErrBadInput
	}

	_, found, err := uc.reports.GetByID(ctx, req.ReportID)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, domain.ErrNotFound
	}
	if req.ActorRole != "moderator" {
		return nil, domain.ErrForbidden
	}

	items, err := uc.attachments.ListByReport(ctx, req.ReportID)
	if err != nil {
		return nil, err
	}
	out := make([]AttachmentWithURL, 0, len(items))
	for _, a := range items {
		signedURL := ""
		if uc.signer != nil {
			signedURL, _ = uc.signer.PresignGetObject(ctx, a.StorageKey, 15*time.Minute)
		}
		out = append(out, AttachmentWithURL{Attachment: a, SignedURL: signedURL})
	}
	return out, nil
}
