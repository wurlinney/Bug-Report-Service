package create_report

import (
	"context"
	"errors"
	"testing"

	"bug-report-service/internal/domain"
	"bug-report-service/internal/domain/report"
)

type mockReportCreator struct {
	result report.Report
	err    error
}

func (m *mockReportCreator) Create(_ context.Context, r report.Report) (report.Report, error) {
	if m.err != nil {
		return report.Report{}, m.err
	}
	out := m.result
	out.ReporterName = r.ReporterName
	out.Description = r.Description
	out.Status = r.Status
	return out, nil
}

type mockSessionGetter struct {
	found bool
	err   error
}

func (m *mockSessionGetter) GetByID(_ context.Context, _ string) (bool, error) {
	return m.found, m.err
}

type mockAttachmentBinder struct {
	err error
}

func (m *mockAttachmentBinder) BindSessionToReport(_ context.Context, _ string, _ string) error {
	return m.err
}

func TestExecute_Success_NoUploadSession(t *testing.T) {
	uc := New(
		&mockReportCreator{result: report.Report{ID: "r1"}},
		&mockSessionGetter{},
		&mockAttachmentBinder{},
	)

	got, err := uc.Execute(context.Background(), Request{
		ReporterName: "Alice",
		Description:  "Something broke",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != "r1" {
		t.Errorf("expected ID r1, got %s", got.ID)
	}
	if got.ReporterName != "Alice" {
		t.Errorf("expected reporter Alice, got %s", got.ReporterName)
	}
	if got.Status != report.StatusNew {
		t.Errorf("expected status %s, got %s", report.StatusNew, got.Status)
	}
}

func TestExecute_Success_WithUploadSession(t *testing.T) {
	uc := New(
		&mockReportCreator{result: report.Report{ID: "r2"}},
		&mockSessionGetter{found: true},
		&mockAttachmentBinder{},
	)

	got, err := uc.Execute(context.Background(), Request{
		ReporterName:    "Bob",
		Description:     "Bug",
		UploadSessionID: "sess-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != "r2" {
		t.Errorf("expected ID r2, got %s", got.ID)
	}
}

func TestExecute_Error_EmptyReporterName(t *testing.T) {
	uc := New(
		&mockReportCreator{},
		&mockSessionGetter{},
		&mockAttachmentBinder{},
	)

	_, err := uc.Execute(context.Background(), Request{
		ReporterName: "   ",
		Description:  "desc",
	})
	if !errors.Is(err, domain.ErrBadInput) {
		t.Fatalf("expected ErrBadInput, got %v", err)
	}
}

func TestExecute_Error_UploadSessionNotFound(t *testing.T) {
	uc := New(
		&mockReportCreator{result: report.Report{ID: "r3"}},
		&mockSessionGetter{found: false},
		&mockAttachmentBinder{},
	)

	_, err := uc.Execute(context.Background(), Request{
		ReporterName:    "Carol",
		Description:     "desc",
		UploadSessionID: "missing-sess",
	})
	if !errors.Is(err, domain.ErrBadInput) {
		t.Fatalf("expected ErrBadInput, got %v", err)
	}
}
