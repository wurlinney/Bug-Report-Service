package list_attachments

import (
	"context"
	"errors"
	"testing"
	"time"

	"bug-report-service/internal/domain"
	"bug-report-service/internal/domain/attachment"
	"bug-report-service/internal/domain/report"
)

type mockReportGetter struct {
	report report.Report
	found  bool
	err    error
}

func (m *mockReportGetter) GetByID(_ context.Context, _ string) (report.Report, bool, error) {
	return m.report, m.found, m.err
}

type mockAttachmentLister struct {
	items []attachment.Attachment
	err   error
}

func (m *mockAttachmentLister) ListByReport(_ context.Context, _ string) ([]attachment.Attachment, error) {
	return m.items, m.err
}

type mockURLSigner struct {
	url string
	err error
}

func (m *mockURLSigner) PresignGetObject(_ context.Context, _ string, _ time.Duration) (string, error) {
	return m.url, m.err
}

func TestExecute_Success(t *testing.T) {
	uc := New(
		&mockReportGetter{found: true},
		&mockAttachmentLister{
			items: []attachment.Attachment{
				{ID: 1, StorageKey: "key1", FileName: "a.png"},
				{ID: 2, StorageKey: "key2", FileName: "b.png"},
			},
		},
		&mockURLSigner{url: "https://signed.url/file"},
	)

	got, err := uc.Execute(context.Background(), Request{
		ActorRole: "moderator",
		ActorID:   "u1",
		ReportID:  "r1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 attachments, got %d", len(got))
	}
	if got[0].SignedURL != "https://signed.url/file" {
		t.Errorf("expected signed URL, got %s", got[0].SignedURL)
	}
}

func TestExecute_Error_ReportNotFound(t *testing.T) {
	uc := New(
		&mockReportGetter{found: false},
		&mockAttachmentLister{},
		&mockURLSigner{},
	)

	_, err := uc.Execute(context.Background(), Request{
		ActorRole: "moderator",
		ActorID:   "u1",
		ReportID:  "missing",
	})
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestExecute_Error_NonModerator(t *testing.T) {
	uc := New(
		&mockReportGetter{found: true},
		&mockAttachmentLister{},
		&mockURLSigner{},
	)

	_, err := uc.Execute(context.Background(), Request{
		ActorRole: "user",
		ActorID:   "u1",
		ReportID:  "r1",
	})
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}
