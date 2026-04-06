package finalize_attachment

import (
	"context"
	"errors"
	"testing"

	"bug-report-service/internal/domain"
	"bug-report-service/internal/domain/attachment"
	"bug-report-service/internal/domain/uploadsession"
)

type mockSessionGetter struct {
	session uploadsession.UploadSession
	found   bool
	err     error
}

func (m *mockSessionGetter) GetByID(_ context.Context, _ string) (uploadsession.UploadSession, bool, error) {
	return m.session, m.found, m.err
}

type mockAttachmentCreator struct {
	result attachment.Attachment
	err    error
}

func (m *mockAttachmentCreator) Create(_ context.Context, a attachment.Attachment) (attachment.Attachment, error) {
	if m.err != nil {
		return attachment.Attachment{}, m.err
	}
	out := m.result
	out.UploadSessionID = a.UploadSessionID
	out.FileName = a.FileName
	out.ContentType = a.ContentType
	out.FileSize = a.FileSize
	out.StorageKey = a.StorageKey
	return out, nil
}

type mockIdempotencyChecker struct {
	att   attachment.Attachment
	found bool
	err   error
}

func (m *mockIdempotencyChecker) GetByIdempotencyKey(_ context.Context, _ string, _ string, _ string) (attachment.Attachment, bool, error) {
	return m.att, m.found, m.err
}

func allowedMIMEs() map[string]struct{} {
	return map[string]struct{}{
		"image/png":       {},
		"image/jpeg":      {},
		"application/pdf": {},
	}
}

func TestExecute_Success(t *testing.T) {
	uc := New(
		&mockSessionGetter{found: true},
		&mockAttachmentCreator{result: attachment.Attachment{ID: 1}},
		&mockIdempotencyChecker{},
		10*1024*1024,
		allowedMIMEs(),
	)

	got, err := uc.Execute(context.Background(), Request{
		UploadSessionID: "sess-1",
		UploadID:        "upload-1",
		FileName:        "screenshot.png",
		ContentType:     "image/png",
		FileSize:        1024,
		StorageKey:      "uploads/screenshot.png",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != 1 {
		t.Errorf("expected ID 1, got %d", got.ID)
	}
	if got.ContentType != "image/png" {
		t.Errorf("expected image/png, got %s", got.ContentType)
	}
}

func TestExecute_Error_EmptySessionID(t *testing.T) {
	uc := New(
		&mockSessionGetter{},
		&mockAttachmentCreator{},
		&mockIdempotencyChecker{},
		10*1024*1024,
		allowedMIMEs(),
	)

	_, err := uc.Execute(context.Background(), Request{
		UploadSessionID: "",
		UploadID:        "upload-1",
		FileName:        "file.png",
		ContentType:     "image/png",
		FileSize:        1024,
		StorageKey:      "key",
	})
	if !errors.Is(err, domain.ErrBadInput) {
		t.Fatalf("expected ErrBadInput, got %v", err)
	}
}

func TestExecute_Error_UnsupportedMIME(t *testing.T) {
	uc := New(
		&mockSessionGetter{found: true},
		&mockAttachmentCreator{},
		&mockIdempotencyChecker{},
		10*1024*1024,
		allowedMIMEs(),
	)

	_, err := uc.Execute(context.Background(), Request{
		UploadSessionID: "sess-1",
		UploadID:        "upload-1",
		FileName:        "virus.exe",
		ContentType:     "application/x-executable",
		FileSize:        1024,
		StorageKey:      "key",
	})
	if !errors.Is(err, domain.ErrBadInput) {
		t.Fatalf("expected ErrBadInput, got %v", err)
	}
}

func TestExecute_Error_SessionNotFound(t *testing.T) {
	uc := New(
		&mockSessionGetter{found: false},
		&mockAttachmentCreator{},
		&mockIdempotencyChecker{},
		10*1024*1024,
		allowedMIMEs(),
	)

	_, err := uc.Execute(context.Background(), Request{
		UploadSessionID: "missing",
		UploadID:        "upload-1",
		FileName:        "file.png",
		ContentType:     "image/png",
		FileSize:        1024,
		StorageKey:      "key",
	})
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
