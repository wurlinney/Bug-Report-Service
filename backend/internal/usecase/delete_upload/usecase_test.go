package delete_upload

import (
	"context"
	"errors"
	"testing"

	"bug-report-service/internal/domain"
)

type mockAttachmentDeleter struct {
	deleted bool
	err     error
}

func (m *mockAttachmentDeleter) DeleteFromSessionByStorageKey(_ context.Context, _ string, _ string) (bool, error) {
	return m.deleted, m.err
}

func TestExecute_Success(t *testing.T) {
	uc := New(&mockAttachmentDeleter{deleted: true})

	ok, err := uc.Execute(context.Background(), "sess-1", "key/file.png")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("expected deleted=true")
	}
}

func TestExecute_Error_EmptySessionID(t *testing.T) {
	uc := New(&mockAttachmentDeleter{})

	_, err := uc.Execute(context.Background(), "", "key/file.png")
	if !errors.Is(err, domain.ErrBadInput) {
		t.Fatalf("expected ErrBadInput, got %v", err)
	}
}
